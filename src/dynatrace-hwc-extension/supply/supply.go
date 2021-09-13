package supply

import (
	// "crypto/md5"

	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudfoundry/libbuildpack"
)

type Stager interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/stager.go
	BuildDir() string
	DepDir() string
	DepsIdx() string
	DepsDir() string
	CacheDir() string
	WriteProfileD(string, string) error
	WriteEnvFile(string, string) error
	/* unused calls
	CacheDir() string
	LinkDirectoryInDepDir(string, string) error
	//AddBinDependencyLink(string, string) error
	WriteEnvFile(string, string) error
	WriteProfileD(string, string) error
	SetStagingEnvironment() error
	*/
}

type TenantInfo struct {
	Tenant         string   `json:"tenantUUID"`
	TenantToken    string   `json:"tenantToken"`
	Communications []string `json:"communicationEndpoints"`
}

type Manifest interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/manifest.go
	AllDependencyVersions(string) []string
	DefaultVersion(string) (libbuildpack.Dependency, error)
}

type Installer interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/installer.go
	InstallDependency(libbuildpack.Dependency, string) error
	InstallOnlyVersion(string, string) error
	/* unused calls
	FetchDependency(libbuildpack.Dependency, string) error
	*/
}

type Command interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/command.go
	Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(dir string, program string, args ...string) (string, error)
	/* unused calls
	Output(string, string, ...string) (string, error)
	*/
}

type Supplier struct {
	Manifest  Manifest
	Installer Installer
	Stager    Stager
	Command   Command
	Log       *libbuildpack.Logger
	/* unused calls
	Config    *config.Config
	Project   *project.Project
	*/
}

// credentials represent the user settings extracted from the environment.
type credentials struct {
	ServiceName       string
	EnvironmentID     string
	CustomOneAgentURL string
	APIToken          string
	PaasToken         string
	APIURL            string
	SkipErrors        bool
	NetworkZone       string
	// DT_CONNECTION_POINT=abc;zdlk;lkfd
	// DT_NETWORK_ZONE
}

const dynatraceAgentFolder = "dynatrace"

var envVars = make(map[string]interface{}, 0)

func (s *Supplier) Run() error {
	s.Log.BeginStep("Supplying Dynatrace HWC Extension")

	s.Log.Info("  >>>>>>> BuildDir: %s", s.Stager.BuildDir())
	s.Log.Info("  >>>>>>> DepDir  : %s", s.Stager.DepDir())
	s.Log.Info("  >>>>>>> DepsIdx : %s", s.Stager.DepsIdx())
	s.Log.Info("  >>>>>>> DepsDir : %s", s.Stager.DepsDir())
	s.Log.Info("  >>>>>>> CacheDir: %s", s.Stager.CacheDir())

	var creds *credentials
	var DTServiceExists bool
	if DTServiceExists, creds = detectDynatraceServices(s); !DTServiceExists {
		s.Log.Info("No Dynatrace service to bind to...")
		return nil
	}

	s.Log.BeginStep("Installing Dynatrace .Net Agent")

	buildpackDir, err := getBuildpackDir(s)
	if err != nil {
		s.Log.Error("Unable to install Dynatrace: %s", err.Error())
		return err
	}
	s.Log.Info("buildpackDir: %v", buildpackDir)

	s.Log.BeginStep("Creating cache directory " + s.Stager.CacheDir())
	if err := os.MkdirAll(s.Stager.CacheDir(), 0755); err != nil {
		s.Log.Error("Failed to create cache directory "+s.Stager.CacheDir(), err)
		return err
	}

	downloadsDir := filepath.Join(s.Stager.DepDir(), "downlaods")

	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		s.Log.Error("Failed to create downloads directory "+downloadsDir, err)
		return err
	}

	dtDownloadLocalFilename := filepath.Join(downloadsDir, "DynatraceAgent.zip")

	s.Log.Info("dtDownloadLocalFilename=" + dtDownloadLocalFilename)
	dtAgentPath := filepath.Join(s.Stager.DepDir(), dynatraceAgentFolder)
	s.Log.Info("Dynatrace Agent Path: " + dtAgentPath)

	dtDownloadURL := getDownloadURL(creds) + "?Api-Token=" + creds.PaasToken

	s.Log.BeginStep("Downloading Dynatrace agent...")
	if err := downloadDependency(s, dtDownloadURL, dtDownloadLocalFilename); err != nil {
		return err
	}

	//listExtractedFiles(s, downloadsDir)
	s.Log.BeginStep("Extracting Dynatrace Agent to %s", dtAgentPath)
	if err := libbuildpack.ExtractZip(dtDownloadLocalFilename, dtAgentPath); err != nil {
		s.Log.Error("Error Extracting Dynatrace Agent", err)
		return err
	}

	// Read tenant, tenanttoken and communications endpooint from the manifest.json file and create standalone.conf file in the agent directory
	if err := createStandaloneFile(s, dtAgentPath); err != nil {
		return err
	}

	// Build procfile so that CF can execute the hwc command
	if err := getProcfile(s, buildpackDir); err != nil {
		return err
	}

	// Build dynatrace.bat file in profile.d directory of the app. This batch file sets all the env variables required for the agent to work
	if err := buildProfileD(s, *creds, dtAgentPath); err != nil {
		return err
	}

	s.Log.Info("Installing Dynatrace Agent Completed.")
	return nil
}

// Detects whether the app is bound to a Dynatrace service or not. When an app is bound to Dynatrace service, VCAP_SERVICES env variable contains
// entry that has dynatrace in it. If this env variable is not found, it is assumed that the app is bound to Dynatrace service.
func detectDynatraceServices(s *Supplier) (bool, *credentials) {
	s.Log.Info("Detecting Dynatrace...")
	var vcapServices map[string][]struct {
		Name        string                 `json:"name"`
		Credentials map[string]interface{} `json:"credentials"`
	}

	if err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices); err != nil {
		s.Log.Info("Failed to unmarshal VCAP_SERVICES: %s", err)
		return false, nil
	}

	var found []*credentials

	for _, services := range vcapServices {
		for _, service := range services {
			s.Log.Info("Service name is " + service.Name)
			if !strings.Contains(strings.ToLower(service.Name), "dynatrace") {
				continue
			}

			queryString := func(key string) string {
				if value, ok := service.Credentials[key].(string); ok {
					return value
				}
				return ""
			}

			creds := &credentials{
				ServiceName:       service.Name,
				EnvironmentID:     queryString("environmentid"),
				APIToken:          queryString("apitoken"),
				APIURL:            queryString("apiurl"),
				CustomOneAgentURL: queryString("customoneagenturl"),
				SkipErrors:        queryString("skiperrors") == "true",
				NetworkZone:       queryString("networkzone"),
				PaasToken:         queryString("paastoken"),
			}

			if (creds.EnvironmentID != "" && creds.PaasToken != "") || creds.CustomOneAgentURL != "" {
				found = append(found, creds)
			} else if creds.EnvironmentID == "" || creds.PaasToken == "" { // One of the fields is empty.
				s.Log.Warning("Incomplete credentials. environment ID: %s, Paas Token: %s",
					creds.EnvironmentID, creds.PaasToken)
			}
		}
	}

	if len(found) >= 1 {
		s.Log.Info("Found one matching service: %s", found[0].ServiceName)
		return true, found[0]
	} else {
		return false, nil
	}
}

// Creates standalone.conf file under agent/conf directory. This file contains tenant, tenanttoken and server entries. The data
// for these entries is read from dynatrace/manifest.json file. This file is required for Paas agent.
func createStandaloneFile(s *Supplier, dtAgentPath string) (err error) {
	var jsonBuffer bytes.Buffer
	manifestFile := filepath.Join(dtAgentPath, "manifest.json")
	mFile, err := os.Open(manifestFile)
	if err != nil {
		s.Log.Error("Error opening manifest.json file")
		return err
	}
	defer mFile.Close()
	byteValue, _ := ioutil.ReadAll(mFile)

	// Reading info from manifest file and writing few entries to the standalone.conf file
	var tenantInfo TenantInfo
	json.Unmarshal(byteValue, &tenantInfo)
	//s.Log.Info("Tenant=" + tenantInfo.Tenant)
	jsonBuffer.WriteString("tenant " + tenantInfo.Tenant + "\n")
	jsonBuffer.WriteString("tenanttoken " + tenantInfo.TenantToken + "\n")

	jsonBuffer.WriteString("server ")
	endpoints := strings.Join(tenantInfo.Communications, ";")
	jsonBuffer.WriteString(endpoints)

	standaloneFile := path.Join(dtAgentPath, "/agent/conf/standalone.conf")
	if err := writeToFile(&jsonBuffer, standaloneFile, 0755); err != nil {
		s.Log.Error("Unable to write to standalone.conf file")
		return err
	}

	return nil
}

// Using libbuildpack utility it gets the buildpack dir name. This directory can later be used to install dependencies from the buildpack
func getBuildpackDir(s *Supplier) (string, error) {
	// get the buildpack directory
	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		s.Log.Error("Unable to determine buildpack directory: %s", err.Error())
	}
	return buildpackDir, err
}

// Given the url and the filepath, this function downloads Dynatrace Paas agent.
func downloadDependency(s *Supplier, url string, filepath string) (err error) {
	//s.Log.Info("Downloading from [%s]", url)
	s.Log.Info("Saving to [%s]", filepath)

	var httpClient = &http.Client{
		Timeout: time.Second * 100,
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return errors.New("bad status: " + resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// Dynatrace download url can be a SaaS url or managed url. This functions look at the entries of credentials and builds the url
func getDownloadURL(c *credentials) string {
	if c.CustomOneAgentURL != "" {
		return c.CustomOneAgentURL
	}

	apiURL := c.APIURL
	if apiURL == "" {
		apiURL = fmt.Sprintf("https://%s.live.dynatrace.com/api", c.EnvironmentID)
	}

	u, err := url.ParseRequestURI(fmt.Sprintf("%s/v1/deployment/installer/agent/windows/paas/latest", apiURL))
	if err != nil {
		return ""
	}

	qv := make(url.Values)
	//qv.Add("bitness", "64")
	// only set the networkzone property when it is configured
	if c.NetworkZone != "" {
		qv.Add("networkzone", c.NetworkZone)
	}
	u.RawQuery = qv.Encode() // Parameters will be sorted by key.

	return u.String()
}

func getProcfile(s *Supplier, buildpackDir string) error {
	procFileBundledWithApp := filepath.Join(s.Stager.BuildDir(), "Procfile")
	procFileBundledWithAppExists, err := libbuildpack.FileExists(procFileBundledWithApp)
	if err != nil {
		// no Procfile found in the app folder
		procFileBundledWithAppExists = false
	}
	if procFileBundledWithAppExists {
		// Procfile exists in app folder
		s.Log.Info("Using Procfile provided in the app folder")
	} else {
		s.Log.Info("No Procfile found in the app folder")
		// looking for Procfile in the buildpack dir
		procFileBundledWithBuildPack := filepath.Join(buildpackDir, "Procfile")
		procFileDest := filepath.Join(s.Stager.BuildDir(), "Procfile")
		procFileBundledWithBuildPackExists, err := libbuildpack.FileExists(procFileBundledWithBuildPack)
		if err != nil {
			s.Log.Error("Error checking if Procfile exists in buildpack", err)
			return err
		}
		if procFileBundledWithBuildPackExists {
			// Procfile exists in buidpack folder
			s.Log.Info("Using Procfile provided with the buildpack")
			if err := libbuildpack.CopyFile(procFileBundledWithBuildPack, procFileDest); err != nil {
				s.Log.Error("Error copying Procfile provided by the buildpack", err)
				return err
			}
			s.Log.Info("Copied Procfile from buildpack to app folder")
		} else {
			s.Log.Info("No Procfile provided by the buildpack")
		}
	}
	return nil
}

func buildProfileD(s *Supplier, cred credentials, dtAgentPath string) error {
	var scriptContentBuffer bytes.Buffer

	s.Log.Info("Setting environment variables for Dynatrace .net agent")

	scriptContentBuffer = setDynatraceProfilerProperties(s, dtAgentPath)

	scriptContent := scriptContentBuffer.String()
	return s.Stager.WriteProfileD("dynatrace.bat", scriptContent)
}

func setDynatraceProfilerProperties(s *Supplier, dtAgentPath string) bytes.Buffer {
	s.Log.Info("Setting Dynatrace profiler properties")
	var profilerSettingsBuffer bytes.Buffer
	profilerSettingsBuffer.WriteString("set COR_ENABLE_PROFILING=1")
	profilerSettingsBuffer.WriteString("\n")
	profilerSettingsBuffer.WriteString("set COR_PROFILER={B7038F67-52FC-4DA2-AB02-969B3C1EDA03}")
	profilerSettingsBuffer.WriteString("\n")
	profilerSettingsBuffer.WriteString("set DT_AGENTACTIVE=true")
	profilerSettingsBuffer.WriteString("\n")
	profilerSettingsBuffer.WriteString("set DT_BLOCKLIST=powershell*")
	profilerSettingsBuffer.WriteString("\n")
	depsDir := filepath.Join("%DEPS_DIR%", s.Stager.DepsIdx())
	agent32bit := filepath.Join(depsDir, "dynatrace\\agent\\lib\\oneagentloader.dll")
	agent64bit := filepath.Join(depsDir, "dynatrace\\agent\\lib64\\oneagentloader.dll")
	profilerSettingsBuffer.WriteString(strings.Join([]string{"set COR_PROFILER_PATH_32=", agent32bit}, ""))
	profilerSettingsBuffer.WriteString("\n")
	profilerSettingsBuffer.WriteString(strings.Join([]string{"set COR_PROFILER_PATH_64=", agent64bit}, ""))
	profilerSettingsBuffer.WriteString("\n")

	return profilerSettingsBuffer
}

func writeToFile(source io.Reader, destFile string, mode os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		return err
	}

	fh, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fh, source)
	if err != nil {
		return err
	}

	return nil
}
