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

* Once the app is pushed, it is important to bind the app to Dynatrace service. When the dynatrace service is being created, it is important that at least these 3 options are set. 
```$xslt
environmentid
paastoken
apitoken
```

