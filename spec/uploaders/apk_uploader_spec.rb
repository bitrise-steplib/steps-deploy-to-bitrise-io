require_relative './../../uploaders/apk_uploader'

describe 'apk_uploader' do
  describe 'filters test' do
    it 'filters package infos' do
      infos = 'package: name=\'hu.kntcrw.cardsup\' versionCode=\'2\' versionName=\'0.9\' platformBuildVersionName=\'6.0-2704002\''

      package_name, version_code, version_name = filter_package_infos(infos)

      expect(package_name).to eq('hu.kntcrw.cardsup')
      expect(version_code).to eq('2')
      expect(version_name).to eq('0.9')
    end

    it 'do not finds package infos' do
      infos = 'hu.kntcrw.cardsup'

      package_name, version_code, version_name = filter_package_infos(infos)

      expect(package_name).to eq('')
      expect(version_code).to eq('')
      expect(version_name).to eq('')
    end

    it 'filters app label' do
      infos = 'application: label=\'CardsUp\' icon=\'res/mipmap-hdpi-v4/ic_launcher.png\''

      app_name = filter_app_label(infos)

      expect(app_name).to eq('CardsUp')
    end

    it 'filters app label' do
      infos = 'application-label:\'CardsUp\''

      app_name = filter_app_label(infos)

      expect(app_name).to eq('CardsUp')
    end

    it 'filters app label' do
      infos = 'application-label:\'CardsUp\''

      app_name = filter_app_label(infos)

      expect(app_name).to eq('CardsUp')
    end

    it 'do not finds app label' do
      infos = 'CardsUp'

      app_name = filter_app_label(infos)

      expect(app_name).to eq('')
    end

    it 'filters min sdk version' do
      infos = 'sdkVersion:\'15\''

      app_name = filter_min_sdk_version(infos)

      expect(app_name).to eq('15')
    end

    it 'do not finds min sdk version' do
      infos = '15'

      app_name = filter_min_sdk_version(infos)

      expect(app_name).to eq('')
    end
  end
end
