require 'net/http'
require 'uri'

# -----------------------
# --- common
# -----------------------

def create_artifact(url, token, file, type)
  file_to_deploy_filename = File.basename(file)

  uri = URI("#{url}/artifacts.json")
  puts "  (i) uri: #{uri}"

  params = {
    'api_token' => token,
    'title' => file_to_deploy_filename,
    'filename' => file_to_deploy_filename,
    'artifact_type' => type
  }

  raw_resp = nil
  (0..2).find { |i|
    if i > 0
      puts "-> Retrying..."
      sleep 5
    end
    begin
      raw_resp = Net::HTTP.post_form(uri, params)
      if raw_resp.code == '200'
        true
      else
        puts " (!) Creat Artifact failed, (response code was: #{raw_resp.code})"
        false
      end
    rescue => ex
      puts " (!) Creat Artifact failed, (exception was: #{ex})"
      false
    end
  }
  fail "Failed to create the Build Artifact on Bitrise" if !raw_resp or raw_resp.code != '200'

  parsed_resp = JSON.parse(raw_resp.body)
  puts "  (i) parsed_resp: #{parsed_resp}"
  fail "Failed to create the Build Artifact on Bitrise: #{parsed_resp['error_msg']}" unless parsed_resp['error_msg'].nil?

  upload_url = parsed_resp['upload_url']
  fail 'No upload_url provided for the artifact' if upload_url.nil?

  artifact_id = parsed_resp['id']
  fail 'No artifact_id provided for the artifact' if artifact_id.nil?

  [upload_url, artifact_id]
end

def upload_file(url, file)
  puts "  (i) upload_url: #{url}"
  is_success = (0..2).find { |i|
    if i > 0
      puts "-> Retrying..."
      sleep 5
    end
    if system("curl --fail --tlsv1 -T '#{file}' -X PUT '#{url}'")
      true
    else
      puts " (!) Upload failed"
      false
    end
  }
  fail 'Failed to upload the Artifact file' if !is_success
end

def finish_artifact(url, token, artifact_id, artifact_info, notify_user_groups, notify_emails, is_enable_public_page)
  uri = URI("#{url}/artifacts/#{artifact_id}/finish_upload.json")
  puts "  (i) uri: #{uri}"

  notify_user_groups = '' if notify_user_groups.to_s == '' || notify_user_groups.to_s == 'none'

  params = { 'api_token' => token }
  params['artifact_info'] = artifact_info unless artifact_info.nil?
  params['notify_user_groups'] = notify_user_groups unless notify_user_groups.nil?
  params['notify_emails'] = notify_emails unless notify_emails.nil?
  params['is_enable_public_page'] = 'yes' if is_enable_public_page

  raw_resp = nil
  (0..2).find { |i|
    if i > 0
      puts "-> Retrying..."
      sleep 5
    end
    begin
      raw_resp = Net::HTTP.post_form(uri, params)
      if raw_resp.code == '200'
        true
      else
        puts " (!) Finish Artifact failed, (response code was: #{raw_resp.code})"
        false
      end
    rescue => ex
      puts " (!) Finish Artifact failed, (exception was: #{ex})"
      false
    end
  }
  fail "Failed to send 'finished' to Bitrise" if !raw_resp or raw_resp.code != '200'

  parsed_resp = JSON.parse(raw_resp.body)
  puts "  (i) parsed_resp: #{parsed_resp}"
  fail 'Failed to send \'finished\' to Bitrise' unless parsed_resp['status'] == 'ok'

  if is_enable_public_page == true
    public_install_page_url = parsed_resp['public_install_page_url']
    raise 'Public Install Page was enabled, but no Public Install Page URL is available!' if public_install_page_url.to_s.empty?
    return public_install_page_url.to_s
  else
    puts 'Public Install Page was disabled, no BITRISE_PUBLIC_INSTALL_PAGE_URL is generated.'
  end
end
