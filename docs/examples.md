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
        ./path/to/test.bundle:TEST_BUNDLE_PATH
```

The Step supports sharing files between pipeline stages. The input needs to be a newline separated **value:env_key_name** list. This metadata will be saved with the individual files and restored by the [Pull Pipeline intermediate files Step](https://www.bitrise.io/integrations/steps/pull-intermediate-files).