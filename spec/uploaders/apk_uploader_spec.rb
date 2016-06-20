require_relative './../../uploaders/apk_uploader'

describe 'apk_uploader' do
  describe 'filters test' do
    it 'it filters package infos' do
      infos = 'package: name=\'hu.kntcrw.cardsup\' versionCode=\'2\' versionName=\'0.9\' platformBuildVersionName=\'6.0-2704002\''

      package_name, version_code, version_name = filter_package_infos(infos)

      expect(package_name).to eq('hu.kntcrw.cardsup')
      expect(version_code).to eq('2')
      expect(version_name).to eq('0.9')
    end

    it 'it filters app label' do
      infos = 'application: label=\'CardsUp\' icon=\'res/mipmap-hdpi-v4/ic_launcher.png\''

      app_name = filter_app_label(infos)

      expect(app_name).to eq('CardsUp')
    end

    it 'it filters min sdk version' do
      infos = 'sdkVersion:\'15\''

      app_name = filter_min_sdk_version(infos)

      expect(app_name).to eq('15')
    end
  end
end
