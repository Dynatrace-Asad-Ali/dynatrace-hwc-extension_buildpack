# dynatrace-hwc-extension-buildpack
---

This buildpack is meant to be used for PCF where the Diego cell is running Windows and the application is written in dotnet.
In order to use this buildpack in your environment, download the zip file and create a buildpack in your PCF env by running the following command:

`cf create-buildpack dynatrace-dotnet-buildpack ./dynatrace-hwc-extension_buildpack-v0.1.zip 99`

