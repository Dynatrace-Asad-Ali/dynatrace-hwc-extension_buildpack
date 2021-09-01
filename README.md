# dynatrace-hwc-extension-buildpack
---

This buildpack is meant to be used for PCF where the Diego cell is running Windows and the application is written in dotnet.
In order to use this buildpack in your environment, download the zip file and create a buildpack in your PCF env by running the following command:

```$xslt
cf create-buildpack dynatrace-dotnet-buildpack ./dynatrace-hwc-extension_buildpack-v0.1.zip 99
```

Once the buildpack is installed, deploy your app with this buildpack.

The following example shows deploying an app using this buildpack
```$xslt
cf push aa_viewenvironment -s windows -f . -b dynatrace-dotnet-buildpack -b hwc_buildpack
```

### Important
* For this buildpack to work, it is very *important* that a manifest file is used to set environment variables. The following environment variables need to be set when an app is pushed.

```$xslt
COR_PROFILER: {B7038F67-52FC-4DA2-AB02-969B3C1EDA03}
COR_ENABLE_PROFILING: 1
DT_AGENTACTIVE: true
DT_BLOCKLIST: powershell*
COR_PROFILER_PATH_32: c:\\Users\\vcap\\app\\dynatrace\\agent\\lib\\oneagentloader.dll
COR_PROFILER_PATH_64: c:\\Users\\vcap\\app\\dynatrace\\agent\\lib64\\oneagentloader.dll
```

* Once the app is pushed, it is important to bind the app to Dynatrace service. When the dynatrace service is being created, it is important that at least these 3 options are set. 
```$xslt
environmentid
paastoken
apitoken
```

