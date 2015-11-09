require 'json'

# -----------------------
# --- upload apk
# -----------------------

def deploy_apk_to_bitrise(apk_path, build_url, api_token, notify_user_groups, notify_emails, is_enable_public_page)
  puts
  puts "# Deploying apk file: #{apk_path}"

  apk_file_size = File.size(apk_path)
  puts "  (i) apk_file_size: #{apk_file_size} KB / #{apk_file_size / 1024.0} MB"

  # - Create a Build Artifact on Bitrise
  puts
  puts '=> Create a Build Artifact on Bitrise'
  upload_url, artifact_id = create_artifact(build_url, api_token, apk_path, 'file')
  fail 'No upload_url provided for the artifact' if upload_url.nil?
  fail 'No artifact_id provided for the artifact' if artifact_id.nil?

  # - Upload the apk
  puts
  puts '=> Upload the apk'
  upload_file(upload_url, apk_path)

  # !!!
  # - Remove this if public page for android implemented on bitrise.io
  # !!!
  if is_enable_public_page
    puts
    puts '(!) Public page is not allowed yet, for apk.'
    puts '(!) is_enable_public_page = false'
    is_enable_public_page = false
  end

  # - Finish the Artifact creation
  puts
  puts '=> Finish the Artifact creation'
  return finish_artifact(build_url,
                         api_token,
                         artifact_id,
                         '',
                         notify_user_groups,
                         notify_emails,
                         is_enable_public_page
                        )
end
