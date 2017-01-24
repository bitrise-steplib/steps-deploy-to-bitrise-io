require 'net/http'
require 'uri'

# -----------------------
# --- common
# -----------------------

def create_artifact(url, token, file, type)
  file_to_deploy_filename = File.basename(file)

  uri = URI("#{url}/artifacts.json")

  file_size_bytes = File.size(file)
  file_size_mb = file_size_bytes.to_f / 1024.0 / 1024.0
  puts " (i) File Size: #{file_size_mb.round(2)} MB"
  params = {
    'api_token' => token,
    'title' => file_to_deploy_filename,
    'filename' => file_to_deploy_filename,
    'artifact_type' => type,
    'file_size_bytes' => file_size_bytes,
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
        puts " (i) uri: #{uri}"
        false
      end
    rescue => ex
      puts " (!) Creat Artifact failed, (exception was: #{ex})"
      puts " (i) uri: #{uri}"
      false
    end
  }
  fail "Failed to create the Build Artifact on Bitrise" if !raw_resp or raw_resp.code != '200'

  parsed_resp = JSON.parse(raw_resp.body)

  unless parsed_resp['error_msg'].nil?
    fail "Failed to create the Build Artifact on Bitrise: #{parsed_resp['error_msg']} | full response was: #{parsed_resp}"
  end

  upload_url = parsed_resp['upload_url']
  fail "No upload_url provided for the artifact | full response was: #{parsed_resp}" if upload_url.nil?

  artifact_id = parsed_resp['id']
  fail "No artifact_id provided for the artifact | full response was: #{parsed_resp}" if artifact_id.nil?

  [upload_url, artifact_id]
end

def upload_file(url, file, content_type=nil)
  curl_call_str = "curl --fail --tlsv1 --globoff"
  if content_type
    curl_call_str = "#{curl_call_str} -H 'Content-Type: #{content_type}'"
  end
  curl_call_str = "#{curl_call_str} -T '#{file}' -X PUT '#{url}'"
  is_success = (0..2).find { |i|
    if i > 0
      puts "-> Retrying..."
      sleep 5
    end
    if system(curl_call_str)
      true
    else
      puts " (!) Upload failed"
      puts " (i) upload_url: #{url}"
      puts " (i) curl call: $ #{curl_call_str}"
      false
    end
  }
  fail 'Failed to upload the Artifact file' if !is_success
end

def finish_artifact(url, token, artifact_id, artifact_info, notify_user_groups, notify_emails, is_enable_public_page)
  uri = URI("#{url}/artifacts/#{artifact_id}/finish_upload.json")

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
        puts " (i) uri: #{uri}"
        false
      end
    rescue => ex
      puts " (!) Finish Artifact failed, (exception was: #{ex})"
      puts " (i) uri: #{uri}"
      false
    end
  }
  fail "Failed to send 'finished' to Bitrise" if !raw_resp or raw_resp.code != '200'

  parsed_resp = JSON.parse(raw_resp.body)
  fail "Failed to send 'finished' to Bitrise | full response was: #{parsed_resp}" unless parsed_resp['status'] == 'ok'

  if is_enable_public_page == true
    public_install_page_url = parsed_resp['public_install_page_url']

    if public_install_page_url.to_s.empty?
      puts " (i) full response was: #{parsed_resp}"
      raise 'Public Install Page was enabled, but no Public Install Page URL is available!'
    end
    return public_install_page_url.to_s
  end

  return ''
end
