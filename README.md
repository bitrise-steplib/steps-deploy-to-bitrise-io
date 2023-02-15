# Deploy to Bitrise.io - Build Artifacts, Test Reports, and Pipeline intermediate files

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-deploy-to-bitrise-io?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-deploy-to-bitrise-io/releases)

Deploys build artifacts to make them available for the user on the build's **Artifacts** tab.  
Sends test results to the Test Reports add-on (build's **Tests** tab),  
and uploads Pipeline intermediate files to make them available in subsequent Stages.

<details>
<summary>Description</summary>

The Step accesses artifacts from a directory specified as the `$BITRISE_DEPLOY_DIR` where artifacts generated by previous Steps gets stored. 
These artifacts are then uploaded on the **Artifacts** tab of any given build. For installable artifacts, such as IPAs or APKs, the Step can create a public install page that allows testers to install the app on their devices. 
You can also use the Step to notify users about the build. If you wish to use the Test Reports add-on, you must add this Step in your Workflow since the Step converts test results to the right format and sends them to the add-on.
The Step can also share Pipeline intermediate files. These files are build artifacts generated by Workflows in a Pipeline intended to be shared with subsequent Stages.

### Configuring the Build Artifact Deployment section of the Step

1. Set the value for the **Deploy directory or file path** required input. The default value is the `$BITRISE_DEPLOY_DIR` Env Var which is exposed by the Bitrise CLI.
If you provide a directory, everything in that directory, excluding sub-directories, gets uploaded. 
If you provide only a file, then only that file gets uploaded. 
To upload a directory's content recursively, you should use the **Compress the artifacts into one file?** which will compress the whole directory, with every sub-directory included.
2. Set the value of the **Notify: User Roles** input. It sends an email with the [public install URL](https://devcenter.bitrise.io/deploy/bitrise-app-deployment/) to those Bitrise users whose roles are included in this field. 
The default value is `everyone`. If you wish to notify based on user roles, add one or more roles and separate them with commas, for example, `developers`, `admins`. If you don't want to notify anyone, set the input to `none`.
3. Set the **Notify: Emails** sensitive input. It sends the public install URL in an email to the email addresses provided here. If you’re adding multiple email address, make sure to separate them with commas. 
The recipients do not have to be in your Bitrise team. Please note that if the email address is associated with a Bitrise account, the user must be [watching](https://devcenter.bitrise.io/builds/configuring-notifications/#watching-an-app) the app.
4. The **Enable public page for the App?** required input is set to `true` by default. It creates a long and random URL which can be shared with those who do not have a Bitrise account. 
If you set this input to `false`, the **Notify: Emails** input will be ignored and the **Notify: User Roles** will receive the build URL instead of the public install URL.
5. With the **Compress the artifacts into one file?** required input set to `true`, you can compress the artifacts found in the Deploy directory into a single file.
You can specify a custom name for the zip file with the `zip_name` option. If you don't specify one, the default `Deploy directory` name will be used. 
If the **Compress the artifacts into one file?** is set to `false`, the artifacts in the Deploy directory will be deployed separately.
6. With the **Format for the BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP output** required input field, you can customize the output format of the public install page’s multiple artifact URLs so that the next Step can render the output (for example, our **Send a Slack message** Step). 
Provide a language template description using [https://golang.org/pkg/text/template](https://golang.org/pkg/text/template) so that the **Deploy to Bitrise.io** Step can build the required custom output.
7. With the **Format for the BITRISE_PERMANENT_DOWNLOAD_URL_MAP output** required input, you can customize the output format of the `BITRISE_PERMANENT_DOWNLOAD_URL_MAP` so that the next Step can render the output.
The next Steps will use this input to generate the related output in the specified format. The output contains multiple permanent URLs for multiple artifacts.
Provide a language template description using [https://golang.org/pkg/text/template](https://golang.org/pkg/text/template) so that the **Deploy to Bitrise.io** Step can build the required custom output.
8. The **Test API's base URL** and the **API Token** input fields are automatically populated for you.

### Configuring the Pipeline Intermediate File Sharing section of the Step

The **Files to share between pipeline stages** input specifies the files meant to be intermediate files shared between the Pipeline Stages. When uploading the Pipeline intermediate files, you must assign environment variable keys to them in the **Files to share between pipeline stages** input.
The inputs `path:env_key` values will be saved together with the file and later automatically reconstructed by the [Pull Pipeline intermediate files Step](https://www.bitrise.io/integrations/steps/pull-intermediate-files).
The directories you specify will be archived and uploaded as a single file.

#### Configuring the Debug section of the Step

If you wish to use any of the Step’s debug features, set the following inputs:
1. In the **Name of the compressed artifact (without .zip extension)** input you can add a custom name for the compressed artifact. If you leave this input empty, the default `Deploy directory` name is used.
Please note that this input only works if you set the **Compress the artifacts into one file?** input to `true`.
2. The **Bitrise Build URL** and the **Bitrise Build API Token** inputs are automatically populated.
3. If **The Enable Debug Mode** required input is set to `true`, the Step prints more verbose logs. It is `false` by default. 
4. If you need a specific [bundletool version](https://github.com/google/bundletool/releases) other than the default value, you can modify the value of the **Bundletool version** required input. 
Bundletool generates an APK from an Android App Bundle so that you can test the APK.

### Troubleshooting

- If your users did not get notified via email, check the **Enable public page for the App?** input. If it is set to `false`, no email notifications will be sent.
- If there are no artifacts uploaded on the **APPS & ARTIFACTS tab**, then check the logs to see if the directory you used in the **Deploy directory or file path** input contained any artifacts. 
- If the email is not received, we recommend, that you check if the email is associated with Bitrise account and if so, if the account is “watching” the app.

### Useful links

- [Deployment on Bitrise](https://devcenter.bitrise.io/deploy/deployment-index/)
- [Watching an app](https://devcenter.bitrise.io/builds/configuring-notifications/#watching-an-app)

### Related Steps

- [Deploy to Google Play](https://www.bitrise.io/integrations/steps/google-play-deploy)
- [Deploy to iTunesConnect](https://www.bitrise.io/integrations/steps/deploy-to-itunesconnect-deliver)
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://devcenter.bitrise.io/steps-and-workflows/steps-and-workflows-index/).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

### Examples

#### Deploy a single file

```yaml
- deploy-to-bitrise-io:
    inputs:
    - deploy_path: /path/to/a/single/file.txt
```

##### Deploy multiple files

```yaml
- deploy-to-bitrise-io:
    inputs:
    - deploy_path: /path/to/a/folder
```

The Step can handle multiple file uploads in one go. In this case the **deploy_path** input has to be a path to a folder.

#### Deploy pipeline intermediate files

```yaml
- deploy-to-bitrise-io:
    inputs:
    - pipeline_intermediate_files: |-
        $BITRISE_IPA_PATH:BITRISE_IPA_PATH
        $BITRISE_APK_PATH:DEVELOPMENT_APK_PATH
        ./path/to/test_reports:TEST_REPORTS_DIR
        $BITRISE_SOURCE_DIR/deploy_dir:DEPLOY_DIR
```

The Step supports sharing files between pipeline stages. The input needs to be a newline separated list of file path - env key pairs (`&lt;path>:&lt;env_key>`).  
This metadata will be saved with the individual files and restored by the [Pull Pipeline intermediate files Step](https://www.bitrise.io/integrations/steps/pull-intermediate-files).


## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `deploy_path` | Specify the directory or file path which will be deployed.  If the specified path is a directory, then every file in the specified directory, excluding sub-directories, will be deployed.  To upload the directory's content recursively, you should use the **Compress the artifacts into one file?** option which compresses the whole directory, with every sub-directory included.  If you specify a file path, then only the specified file will be deployed.  |  | `$BITRISE_DEPLOY_DIR` |
| `is_compress` | If this option is set to `true` and a Deploy directory was specified,  the artifacts in that directory will be compressed into a single ZIP file.  You can specify a custom name for the ZIP using the `zip_name` option. If you do not specify a custom name, the default `Deploy directory` name will be used.  If this option is set to `false`, the artifacts found in the Deploy directory folder will be deployed separately. | required | `false` |
| `zip_name` | If you do not specify a custom name, the Deploy directory name will be used. You can specify a custom name for the ZIP using the `zip_name` option.   This option only works if you selected *true* for *is_compress*. |  |  |
| `notify_user_groups` | Your App's user roles you want to notify. Separate the role names with commas. Possible role names:  * none * testers * developers * admins * owner * everyone  An example to notify your developers and testers:  `testers, developers`  If you want to notify everyone in the app's team, just specify `everyone`.  If you don't want to notify anyone, set this to `none`.  |  | `everyone` |
| `notify_email_list` | Email addresses to notify. Separate them with commas.  You can specify any email address, the recipients don't have to be in your team.  Please note that if the email address is associated with a Bitrise account,  the user must be [watching](https://devcenter.bitrise.io/builds/configuring-notifications/#watching-an-app) the app.  | sensitive |  |
| `is_enable_public_page` | If this option is enabled, a public install page will be available with a long and random URL which can be shared with others who are not registered on Bitrise.  If you disable this option, the **Notify: Emails** option will be ignored and the **Notify: User Roles** users will receive the build's URL instead of the public page's URL!  | required | `true` |
| `bundletool_version` | If you need a specific [bundletool version]((https://github.com/google/bundletool/releases) other than the default version,   you can modify the value of the **Bundletool version** required input. | required | `1.8.1` |
| `build_url` | Unique build URL of this build on Bitrise.io | required | `$BITRISE_BUILD_URL` |
| `build_api_token` | The build's API Token for the build on Bitrise.io | required, sensitive | `$BITRISE_BUILD_API_TOKEN` |
| `pipeline_intermediate_files` | A newline separated list of file path - env key pairs (&lt;path>:&lt;env\_key>).  The file path can be specified with environment variables or direct paths,   and can point to both a local file or directory: ``` $BITRISE_IPA_PATH:BITRISE_IPA_PATH $BITRISE_APK_PATH:DEVELOPMENT_APK_PATH ./path/to/test_reports:TEST_REPORTS_DIR $BITRISE_SOURCE_DIR/deploy_dir:DEPLOY_DIR ``` |  |  |
| `addon_api_base_url` | The URL where test API is accessible.  | required | `https://vdt.bitrise.io/test` |
| `addon_api_token` | The token required to authenticate with the API.  | sensitive | `$ADDON_VDTESTING_API_TOKEN` |
| `public_install_page_url_map_format` | Provide a language template description using [Golang templates](https://golang.org/pkg/text/template) so that the **Deploy to Bitrise.io** Step can build the required custom output. | required | `{{range $index, $element := .}}{{if $index}}\|{{end}}{{$element.File}}=>{{$element.URL}}{{end}}` |
| `permanent_download_url_map_format` | Provide a language template description using [Golang templates](https://golang.org/pkg/text/template) so that the **Deploy to Bitrise.io** Step can build the required custom output for the permanent download URL.   | required | `{{range $index, $element := .}}{{if $index}}\|{{end}}{{$element.File}}=>{{$element.URL}}{{end}}` |
| `debug_mode` | The Step will print more verbose logs if enabled. | required | `false` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `BITRISE_PUBLIC_INSTALL_PAGE_URL` | Public Install Page's URL, if the *Enable public page for the App?* option was *enabled*. |
| `BITRISE_PUBLIC_INSTALL_PAGE_URL_MAP` | Public Install Page URLs by the artifact's file path. Only set it if the *Enable public page for the App?* option was *enabled*.  The default format is `KEY1=>VALUE\|KEY2=>VALUE` but is controlled by the `public_install_page_url_map_format` input  Examples:  - $BITRISE_DEPLOY_DIR/ios_app.ipa=>https://ios_app/public/install/page - $BITRISE_DEPLOY_DIR/android_app.apk=>https://android_app/public/install/page\|$BITRISE_DEPLOY_DIR/ios_app.ipa=>https://ios_app/public/install/page |
| `BITRISE_PERMANENT_DOWNLOAD_URL_MAP` | The output contains permanent Download URLs for each artifact. The URLs can be shared in any communication channel and they won't expire. The default format is `KEY1=>VALUE\|KEY2=>VALUE` where key is the filename and the value is the URL. If you change `permanent_download_url_map_format` input then that will modify the format of this Env Var. You can customize the format of the multiple URLs.  Examples:  - $BITRISE_DEPLOY_DIR/ios_app.ipa=>https://app.bitrise.io/artifacts/ipa-slug/download - $BITRISE_DEPLOY_DIR/android_app.apk=>https://app.bitrise.io/artifacts/apk-slug/download\|$BITRISE_DEPLOY_DIR/ios_app.ipa=>https://app.bitrise.io/artifacts/ipa-slug/download |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-deploy-to-bitrise-io/pulls) and [issues](https://github.com/bitrise-steplib/steps-deploy-to-bitrise-io/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://devcenter.bitrise.io/bitrise-cli/run-your-first-build/).

Learn more about developing steps:

- [Create your own step](https://devcenter.bitrise.io/contributors/create-your-own-step/)
- [Testing your Step](https://devcenter.bitrise.io/contributors/testing-and-versioning-your-steps/)
