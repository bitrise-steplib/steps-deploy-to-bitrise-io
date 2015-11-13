require 'json'

def fail_with_message(message)
  puts "\e[31m#{message}\e[0m"
  exit(1)
end

def build_tool_version_greater?(version, compare_version)
  version_componts = version.split('.')
  compare_version_components = compare_version.split('.')

  if version_componts.count != 3 || compare_version_components.count != 3
    fail_with_message("Invalid build-tool version #{version} | #{compare_version}")
  end

  for i in 0..version_componts.count - 1
    next if compare_version_components[i].to_i == version_componts[i].to_i
    return false if compare_version_components[i].to_i < version_componts[i].to_i
    return true if compare_version_components[i].to_i > version_componts[i].to_i
  end
  return false
end

def aapt_path
  android_home = ENV['ANDROID_HOME']
  if android_home.nil? || android_home == ''
    fail_with_message('Failed to get ANDROID_HOME env')
  end

  aapt_files = Dir[File.join(android_home, 'build-tools', '/**/aapt')]
  fail_with_message('Failed to find aapt tool') unless aapt_files

  latest_build_tool_version = ''
  latest_aapt_path = ''
  aapt_files.each do |aapt_file|
    path_splits = aapt_file.to_s.split('/')
    build_tool_version = path_splits[path_splits.count - 2]

    latest_build_tool_version = build_tool_version if latest_build_tool_version == ''
    if build_tool_version_greater?(latest_build_tool_version, build_tool_version)
      latest_build_tool_version = build_tool_version
      latest_aapt_path = aapt_file.to_s
    end
  end

  fail_with_message('Failed to find latest aapt tool') if latest_aapt_path == ''
  return latest_aapt_path
end

# -----------------------
# --- upload apk
# -----------------------

def deploy_apk_to_bitrise(apk_path, build_url, api_token, notify_user_groups, notify_emails, is_enable_public_page)
  puts
  puts "# Deploying apk file: #{apk_path}"

  # - Analyze the apk / collect infos from apk
  puts
  puts '=> Analyze the apk'

  aapt = aapt_path
  infos = `#{aapt} dump badging #{apk_path}`

  package_name_version_regex = 'package: name=\'(?<package_name>.*)\' versionCode=\'(?<version_code>.*)\' versionName=\'(?<version_name>.*)\' '
  package_name_version_match = infos.match(package_name_version_regex)
  package_name = package_name_version_match.captures[0] if package_name_version_match && package_name_version_match.captures
  version_code = package_name_version_match.captures[1] if package_name_version_match && package_name_version_match.captures
  version_name = package_name_version_match.captures[2] if package_name_version_match && package_name_version_match.captures

  app_name_regex = 'application-label:\'(?<min_sdk_version>.*)\''
  app_name_match = infos.match(app_name_regex)
  app_name = app_name_match.captures[0] if app_name_match && app_name_match.captures

  min_sdk_regex = 'sdkVersion:\'(?<min_sdk_version>.*)\''
  min_sdk_match = infos.match(min_sdk_regex)
  min_sdk = min_sdk_match.captures[0] if min_sdk_match && min_sdk_match.captures

  apk_file_size = File.size(apk_path)

  apk_info_hsh = {
    file_size_bytes: apk_file_size,
    app_info: {
      app_name: app_name,
      package_name: package_name,
      version_code: version_code,
      version_name: version_name,
      min_sdk_version: min_sdk
    }
  }
  puts "  (i) apk_info_hsh: #{apk_info_hsh}"

  # - Create a Build Artifact on Bitrise
  puts
  puts '=> Create a Build Artifact on Bitrise'
  upload_url, artifact_id = create_artifact(build_url, api_token, apk_path, 'android-apk')
  fail 'No upload_url provided for the artifact' if upload_url.nil?
  fail 'No artifact_id provided for the artifact' if artifact_id.nil?

  # - Upload the apk
  puts
  puts '=> Upload the apk'
  upload_file(upload_url, apk_path)

  # - Finish the Artifact creation
  puts
  puts '=> Finish the Artifact creation'
  return finish_artifact(build_url,
                         api_token,
                         artifact_id,
                         JSON.dump(apk_info_hsh),
                         notify_user_groups,
                         notify_emails,
                         is_enable_public_page
                        )
end
