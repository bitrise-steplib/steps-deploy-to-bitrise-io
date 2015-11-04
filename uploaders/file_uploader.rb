require_relative 'common'

# -----------------------
# --- upload file
# -----------------------

def deploy_file_to_bitrise(file_path, build_url, api_token)
  puts
  puts "# Deploying file: #{file_path}"

  # - Create a Build Artifact on Bitrise
  puts
  puts '=> Create a Build Artifact on Bitrise'
  upload_url, artifact_id = create_artifact(build_url, api_token, file_path, 'file')
  fail 'No upload_url provided for the artifact' if upload_url.nil?
  fail 'No artifact_id provided for the artifact' if artifact_id.nil?

  # - Upload the file
  puts
  puts '=> Upload the file'
  upload_file(upload_url, file_path)

  # - Finish the Artifact creation
  puts
  puts '=> Finish the Artifact creation'
  finish_artifact(build_url,
                  api_token,
                  artifact_id,
                  '',
                  '',
                  '',
                  false
                 )
end
