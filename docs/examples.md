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
        BITRISE_IPA_PATH
        $BITRISE_APK_PATH:DEVELOPMENT_APK_PATH
        ./path/to/test_reports:TEST_REPORTS_DIR
        $BITRISE_SOURCE_DIR/deploy_dir:DEPLOY_DIR
```

The Step supports sharing files between pipeline stages. The input needs to be a newline (`\n`) separated list of file path - env key pairs (`{path}:{env_key}`).
A shorthand syntax is available when the env var names are the same, e.g. `BITRISE_IPA_PATH` instead of `$BITRISE_IPA_PATH:BITRISE_IPA_PATH`. 
This metadata will be saved with the individual files and restored by the [Pull Pipeline intermediate files Step](https://www.bitrise.io/integrations/steps/pull-intermediate-files).
