package uploaders

import "testing"

func Test_filterPackageInfos(t *testing.T) {

	tests := []struct {
		name    string
		aaptOut string
		want    string
		want1   string
		want2   string
	}{
		{
			name:    "test_with_empty_platformBuildVersionName",
			aaptOut: aaptOutEmptyPlatformBuildVersionName,
			want:    "com.example.birmachera.myapplication",
			want1:   "1",
			want2:   "1.0",
		},
		{
			name:    "test_without_platformBuildVersionName",
			aaptOut: `package: name='com.example.birmachera.myapplication' versionCode='1' versionName='1.0'`,
			want:    "com.example.birmachera.myapplication",
			want1:   "1",
			want2:   "1.0",
		},
		{
			name:    "test_with_platformBuildVersionName",
			aaptOut: `package: name='com.example.birmachera.myapplication' versionCode='1' versionName='1.0'  platformBuildVersionName='3'`,
			want:    "com.example.birmachera.myapplication",
			want1:   "1",
			want2:   "1.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := filterPackageInfos(tt.aaptOut)
			if got != tt.want {
				t.Errorf("filterPackageInfos() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("filterPackageInfos() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("filterPackageInfos() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

var aaptOutEmptyPlatformBuildVersionName = `package: name='com.example.birmachera.myapplication' versionCode='1' versionName='1.0' platformBuildVersionName=''
sdkVersion:'17'
targetSdkVersion:'28'
uses-permission: name='android.permission.INTERNET'
application-label:'My Application'
application-label-af:'My Application'
application-label-am:'My Application'
application-label-ar:'My Application'
application-label-as:'My Application'
application-label-az:'My Application'
application-label-be:'My Application'
application-label-bg:'My Application'
application-label-bn:'My Application'
application-label-bs:'My Application'
application-label-ca:'My Application'
application-label-cs:'My Application'
application-label-da:'My Application'
application-label-de:'My Application'
application-label-el:'My Application'
application-label-en-AU:'My Application'
application-label-en-CA:'My Application'
application-label-en-GB:'My Application'
application-label-en-IN:'My Application'
application-label-en-XC:'My Application'
application-label-es:'My Application'
application-label-es-US:'My Application'
application-label-et:'My Application'
application-label-eu:'My Application'
application-label-fa:'My Application'
application-label-fi:'My Application'
application-label-fr:'My Application'
application-label-fr-CA:'My Application'
application-label-gl:'My Application'
application-label-gu:'My Application'
application-label-hi:'My Application'
application-label-hr:'My Application'
application-label-hu:'My Application'
application-label-hy:'My Application'
application-label-in:'My Application'
application-label-is:'My Application'
application-label-it:'My Application'
application-label-iw:'My Application'
application-label-ja:'My Application'
application-label-ka:'My Application'
application-label-kk:'My Application'
application-label-km:'My Application'
application-label-kn:'My Application'
application-label-ko:'My Application'
application-label-ky:'My Application'
application-label-lo:'My Application'
application-label-lt:'My Application'
application-label-lv:'My Application'
application-label-mk:'My Application'
application-label-ml:'My Application'
application-label-mn:'My Application'
application-label-mr:'My Application'
application-label-ms:'My Application'
application-label-my:'My Application'
application-label-nb:'My Application'
application-label-ne:'My Application'
application-label-nl:'My Application'
application-label-or:'My Application'
application-label-pa:'My Application'
application-label-pl:'My Application'
application-label-pt:'My Application'
application-label-pt-BR:'My Application'
application-label-pt-PT:'My Application'
application-label-ro:'My Application'
application-label-ru:'My Application'
application-label-si:'My Application'
application-label-sk:'My Application'
application-label-sl:'My Application'
application-label-sq:'My Application'
application-label-sr:'My Application'
application-label-sr-Latn:'My Application'
application-label-sv:'My Application'
application-label-sw:'My Application'
application-label-ta:'My Application'
application-label-te:'My Application'
application-label-th:'My Application'
application-label-tl:'My Application'
application-label-tr:'My Application'
application-label-uk:'My Application'
application-label-ur:'My Application'
application-label-uz:'My Application'
application-label-vi:'My Application'
application-label-zh-CN:'My Application'
application-label-zh-HK:'My Application'
application-label-zh-TW:'My Application'
application-label-zu:'My Application'
application-icon-120:'res/mipmap-anydpi-v26/ic_launcher.xml'
application-icon-160:'res/mipmap-anydpi-v26/ic_launcher.xml'
application-icon-240:'res/mipmap-anydpi-v26/ic_launcher.xml'
application-icon-320:'res/mipmap-anydpi-v26/ic_launcher.xml'
application-icon-480:'res/mipmap-anydpi-v26/ic_launcher.xml'
application-icon-640:'res/mipmap-anydpi-v26/ic_launcher.xml'
application-icon-65534:'res/mipmap-anydpi-v26/ic_launcher.xml'
application: label='My Application' icon='res/mipmap-anydpi-v26/ic_launcher.xml'
application-debuggable
launchable-activity: name='com.example.birmachera.myapplication.MainActivity'  label='' icon=''
feature-group: label=''
  uses-feature: name='android.hardware.faketouch'
  uses-implied-feature: name='android.hardware.faketouch' reason='default feature for all apps'
main
supports-screens: 'small' 'normal' 'large' 'xlarge'
supports-any-density: 'true'
locales: '--_--' 'af' 'am' 'ar' 'as' 'az' 'be' 'bg' 'bn' 'bs' 'ca' 'cs' 'da' 'de' 'el' 'en-AU' 'en-CA' 'en-GB' 'en-IN' 'en-XC' 'es' 'es-US' 'et' 'eu' 'fa' 'fi' 'fr' 'fr-CA' 'gl' 'gu' 'hi' 'hr' 'hu' 'hy' 'in' 'is' 'it' 'iw' 'ja' 'ka' 'kk' 'km' 'kn' 'ko' 'ky' 'lo' 'lt' 'lv' 'mk' 'ml' 'mn' 'mr' 'ms' 'my' 'nb' 'ne' 'nl' 'or' 'pa' 'pl' 'pt' 'pt-BR' 'pt-PT' 'ro' 'ru' 'si' 'sk' 'sl' 'sq' 'sr' 'sr-Latn' 'sv' 'sw' 'ta' 'te' 'th' 'tl' 'tr' 'uk' 'ur' 'uz' 'vi' 'zh-CN' 'zh-HK' 'zh-TW' 'zu'
densities: '120' '160' '240' '320' '480' '640' '65534'`
