#!/bin/bash

THIS_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

set -e
set -x

cd "$THIS_SCRIPT_DIR"

bundle install

bundle exec ruby "./step.rb" \
  -u "${build_url}" \
  -t "${build_api_token}" \
  -c "${is_compress}" \
  -d "${deploy_path}" \
  -g "${notify_user_groups}" \
  -e "${notify_email_list}" \
  -p "${is_enable_public_page}"
