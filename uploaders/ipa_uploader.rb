require 'json'
require 'ipa_analyzer'
require_relative 'common'

# -----------------------
# --- upload ipa
# -----------------------

def deploy_ipa_to_bitrise(ipa_path, build_url, api_token, notify_user_groups, notify_emails, is_enable_public_page)
  puts
  puts "# Deploying ipa file: #{ipa_path}"

  # - Analyze the IPA / collect infos from IPA
  puts
  puts '=> Analyze the IPA'

  parsed_ipa_infos = {
    mobileprovision: nil,
    info_plist: nil
  }
  ipa_analyzer = IpaAnalyzer::Analyzer.new(ipa_path)
  begin
    puts '  => Opening the IPA'
    ipa_analyzer.open!

    puts '  => Collecting Provisioning Profile information'
    parsed_ipa_infos[:mobileprovision] = ipa_analyzer.collect_provision_info
    fail 'Failed to collect Provisioning Profile information' if parsed_ipa_infos[:mobileprovision].nil?

    puts '  => Collecting Info.plist information'
    parsed_ipa_infos[:info_plist] = ipa_analyzer.collect_info_plist_info
    fail 'Failed to collect Info.plist information' if parsed_ipa_infos[:info_plist].nil?
  rescue => ex
    puts
    puts "Failed: #{ex}"
    puts
    raise ex
  ensure
    puts '  => Closing the IPA'
    ipa_analyzer.close
  end
  puts
  puts '  (i) Parsed IPA infos:'
  puts parsed_ipa_infos
  puts

  ipa_file_size = File.size(ipa_path)
  puts "  (i) ipa_file_size: #{ipa_file_size} KB / #{ipa_file_size / 1024.0} MB"

  info_plist_content = parsed_ipa_infos[:info_plist][:content]
  mobileprovision_content = parsed_ipa_infos[:mobileprovision][:content]
  ipa_info_hsh = {
    file_size_bytes: ipa_file_size,
    app_info: {
      app_title: info_plist_content['CFBundleName'],
      bundle_id: info_plist_content['CFBundleIdentifier'],
      version: info_plist_content['CFBundleShortVersionString'],
      build_number: info_plist_content['CFBundleVersion'],
      min_OS_version: info_plist_content['MinimumOSVersion'],
      device_family_list: info_plist_content['UIDeviceFamily']
    },
    provisioning_info: {
      creation_date: mobileprovision_content['CreationDate'],
      expire_date: mobileprovision_content['ExpirationDate'],
      device_UDID_list: mobileprovision_content['ProvisionedDevices'],
      team_name: mobileprovision_content['TeamName'],
      profile_name: mobileprovision_content['Name'],
      provisions_all_devices: mobileprovision_content['ProvisionsAllDevices']
    }
  }
  puts "  (i) ipa_info_hsh: #{ipa_info_hsh}"

  # - Create a Build Artifact on Bitrise
  puts
  puts '=> Create a Build Artifact on Bitrise'
  upload_url, artifact_id = create_artifact(build_url, api_token, ipa_path, 'ios-ipa')
  fail 'No upload_url provided for the artifact' if upload_url.nil?
  fail 'No artifact_id provided for the artifact' if artifact_id.nil?

  # - Upload the IPA
  puts
  puts '=> Upload the ipa'
  upload_file(upload_url, ipa_path)

  # - Finish the Artifact creation
  puts
  puts '=> Finish the Artifact creation'
  finish_artifact(build_url,
                  api_token,
                  artifact_id,
                  JSON.dump(ipa_info_hsh),
                  notify_user_groups,
                  notify_emails,
                  is_enable_public_page
                 )
end
