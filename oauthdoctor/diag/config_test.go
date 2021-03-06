// Copyright 2019 Google LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package diag

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
)

func TestValidate(t *testing.T) {
	const goodDevToken = "_asdfbasd_-0adsfaw8762"
	const goodClientID = "012345678-8hafs7yfas0f0fh.apps.googleusercontent.com"
	const goodToken = "89yashfoasuf0ujafi0f"
	const goodSecret = "09aufj0aj0ufa8s"

	tests := []struct {
		desc   string
		cfg    ConfigFile
		want   bool
		errstr string
	}{
		{
			desc: "Everything passes",
			cfg: ConfigFile{
				OAuthType: InstalledApp,
				ConfigKeys: ConfigKeys{
					DevToken:        goodDevToken,
					ClientID:        goodClientID,
					ClientSecret:    goodSecret,
					RefreshToken:    goodToken,
					LoginCustomerID: "1111111111",
				},
			},
			want:   true,
			errstr: "nil",
		},
		{
			desc: "Invalid DevToken",
			cfg: ConfigFile{
				OAuthType: InstalledApp,
				ConfigKeys: ConfigKeys{
					DevToken:     "INSERT_DEV_TOKEN_HERE",
					ClientID:     goodClientID,
					ClientSecret: goodSecret,
					RefreshToken: goodToken,
				},
			},
			want:   false,
			errstr: "DevToken",
		},
		{
			desc: "Invalid Client",
			cfg: ConfigFile{
				OAuthType: InstalledApp,
				ConfigKeys: ConfigKeys{
					DevToken:     goodDevToken,
					ClientID:     "randomClientID",
					ClientSecret: goodSecret,
					RefreshToken: goodToken,
				},
			},
			want:   false,
			errstr: "ClientID",
		},
		{
			desc: "Installed App flow: Missing a required key",
			cfg: ConfigFile{
				OAuthType: InstalledApp,
				ConfigKeys: ConfigKeys{
					DevToken:     goodDevToken,
					ClientID:     goodClientID,
					RefreshToken: goodToken,
				},
			},
			want:   false,
			errstr: "ClientSecret",
		},
		{
			desc: "Service account flow: Missing a required key",
			cfg: ConfigFile{
				OAuthType: ServiceAccount,
				ConfigKeys: ConfigKeys{
					DevToken:       goodDevToken,
					PrivateKeyPath: "GoodPath",
				},
			},
			want:   false,
			errstr: "DelegatedAccount",
		},
		{
			desc: "LoginCustomerID cannot have dashes",
			cfg: ConfigFile{
				OAuthType: InstalledApp,
				ConfigKeys: ConfigKeys{
					LoginCustomerID: "111-111-1111",
				},
			},
			want:   false,
			errstr: "LoginCustomerID",
		},
	}

	for _, test := range tests {
		got, err := test.cfg.Validate()
		if got != test.want || !strings.Contains(errstring(err), test.errstr) {
			t.Errorf("%s\ngot: %+v\nwant: %+v\nError: %s, but missing %s in error msg",
				test.desc, got, test.want, errstring(err), test.errstr)
		}
	}
}

func TestGetConfigFile(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		t.Errorf("Error getting current user: %s\n", err)
	}

	tests := []struct {
		desc     string
		lang     string
		filepath string
		want     ConfigFile
	}{
		{
			desc: "(Python) Get default config file",
			lang: "python",
			want: ConfigFile{
				Filename: "google-ads.yaml",
				Filepath: usr.HomeDir,
				Lang:     "python",
			},
		},
		{
			desc: "(Ruby) Get default config file",
			lang: "ruby",
			want: ConfigFile{
				Filename: "google_ads_config.rb",
				Filepath: usr.HomeDir,
				Lang:     "ruby",
			},
		},
		{
			desc: "(.NET) Get default config file",
			lang: "dotnet",
			want: ConfigFile{
				Filename: "App.Config",
				Filepath: usr.HomeDir,
				Lang:     "dotnet",
			},
		},
		{
			desc: "(PHP) Get default config file",
			lang: "php",
			want: ConfigFile{
				Filename: "google_ads_php.ini",
				Filepath: usr.HomeDir,
				Lang:     "php",
			},
		},
		{
			desc: "(Java) Get default config file",
			lang: "java",
			want: ConfigFile{
				Filename: "ads.properties",
				Filepath: usr.HomeDir,
				Lang:     "java",
			},
		},
		{
			desc:     "(Java) Get config file by given path",
			lang:     "java",
			filepath: "/random/config/filepath",
			want: ConfigFile{
				Filename: "filepath",
				Filepath: "/random/config",
				Lang:     "java",
			},
		},
	}

	for _, test := range tests {
		got := GetConfigFile(test.lang, test.filepath)

		if got != test.want {
			t.Errorf("%s\ngot: %s\nwant: %s", test.desc, got, test.want)
		}
	}
}

func TestPrint(t *testing.T) {
	tests := []struct {
		desc    string
		cfg     ConfigFile
		hidePII bool
		want    string
	}{
		{
			desc: "Print out sensitive info",
			cfg: ConfigFile{
				ConfigKeys: ConfigKeys{
					ClientID: "someClientID",
				},
			},
			hidePII: false,
			want:    "someClientID",
		},
		{
			desc: "Print non-sensitive info with hidePII=true",
			cfg: ConfigFile{
				ConfigKeys: ConfigKeys{
					LoginCustomerID: "1234567890",
				},
			},
			hidePII: true,
			want:    "1234567890",
		},
		{
			desc: "Hide sensitive info in config file",
			cfg: ConfigFile{
				ConfigKeys: ConfigKeys{
					ClientID: "someClientID",
				},
			},
			hidePII: true,
			want:    "******",
		},
		{
			desc: "Print service account info",
			cfg: ConfigFile{
				OAuthType: ServiceAccount,
				ServiceAccountInfo: ServiceAccountInfo{
					PrivateKeyID: "someKeyID",
				},
			},
			hidePII: false,
			want:    "someKeyID",
		},
		{
			desc: "Hide sensitive service account info",
			cfg: ConfigFile{
				OAuthType: ServiceAccount,
				ServiceAccountInfo: ServiceAccountInfo{
					PrivateKeyID: "someKeyID",
				},
			},
			hidePII: true,
			want:    "******",
		},
	}

	for _, test := range tests {
		output := new(bytes.Buffer)
		log.SetOutput(output)
		test.cfg.Print(test.hidePII)

		if !strings.Contains(output.String(), test.want) {
			t.Errorf("%s\ngot: %s\nwant substring: %s", test.desc, output, test.want)
		}
	}
}

func TestReplaceConfig(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	now := time.Now().Format("2006-01-02_")
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current dir: %s", err)
	}

	test := struct {
		desc   string
		cfg    ConfigFile
		backup string
	}{
		desc: "Config file and backup file exist",
		cfg: ConfigFile{
			Filepath: filepath.Join(dir, "testdata"),
			Filename: "python_config2",
			Lang:     "python",
		},
		backup: "diag/testdata/python_config2_" + now,
	}

	backup := test.cfg.ReplaceConfig(DevToken, "randomToken")
	config := test.cfg.GetFilepath()

	defer func() {
		// Cleaning up files
		if err = os.Remove(config); err != nil {
			t.Fatalf("%s\nError cleaning up the new config file (%s): %s", test.desc, config, err)
		}

		if err = os.Rename(backup, config); err != nil {
			t.Fatalf("%s\nError renaming the config file from %s to %s: %s", test.desc, backup, config, err)
		}
	}()

	if _, err = os.Stat(config); err != nil {
		t.Fatalf("%s\nProblem finding the config file (%s): %s", test.desc, config, err)
	}

	if !strings.Contains(backup, test.backup) {
		t.Errorf("%s\nBackup config file (%s) is missing and expecting %s: %s", test.desc, backup, test.backup, err)
	}
}

func TestReplaceConfigFromReader(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting current dir: %s", err)
	}

	tests := []struct {
		desc      string
		key       string
		val       string
		cfg       ConfigFile
		commented string
		added     string
	}{
		{
			desc: "(Python) Replace refresh token correctly",
			key:  RefreshToken,
			val:  "new_refresh_token",
			cfg: ConfigFile{
				Lang:     "python",
				Filepath: filepath.Join(dir, "testdata"),
				Filename: "python_config",
			},
			commented: "#refresh_token: 1/PG1",
			added:     "\nrefresh_token:new_refresh_token",
		},
		{
			desc: "(Ruby) Replace client ID correctly",
			key:  ClientID,
			val:  "new_client_id",
			cfg: ConfigFile{
				Lang:     "ruby",
				Filepath: filepath.Join(dir, "testdata"),
				Filename: "ruby_config",
			},
			commented: "#c.client_id = 'GoodClientID'",
			added:     "\nc.client_id= \"new_client_id\"",
		},
		{
			desc: "(.NET) Replace dev token correctly",
			key:  DevToken,
			val:  "new_dev_token",
			cfg: ConfigFile{
				Lang:     "dotnet",
				Filepath: filepath.Join(dir, "testdata"),
				Filename: "dotnet_config1",
			},
			commented: "<!--<add key=\"DeveloperToken\" value=\"GoodDevToken\"/>-->",
			added:     "\n<add key=\"DeveloperToken\" value=\"new_dev_token\"/>",
		},
		{
			desc: "(.NET) Add a dev token without replacing",
			key:  DevToken,
			val:  "new_dev_token",
			cfg: ConfigFile{
				Lang:     "dotnet",
				Filepath: filepath.Join(dir, "testdata"),
				Filename: "dotnet_config3",
			},
			commented: "",
			added:     "\n<add key=\"DeveloperToken\" value=\"new_dev_token\"/>",
		},
		{
			desc: "(PHP) Replace client secret correctly",
			key:  ClientSecret,
			val:  "new_client_secret",
			cfg: ConfigFile{
				Lang:     "php",
				Filepath: filepath.Join(dir, "testdata"),
				Filename: "php_config",
			},
			commented: ";clientSecret = \"GoodClientSecret\"",
			added:     "\nclientSecret= \"new_client_secret\"",
		},
		{
			desc: "(Java) Replace refresh token correctly",
			key:  RefreshToken,
			val:  "new_refresh_token",
			cfg: ConfigFile{
				Lang:     "java",
				Filepath: filepath.Join(dir, "testdata"),
				Filename: "java_config",
			},
			commented: "#api.googleads.refreshToken=",
			added:     "\napi.googleads.refreshToken=new_refresh_token",
		},
	}

	for _, test := range tests {
		f, err := os.Open(test.cfg.GetFilepath())
		if err != nil {
			t.Fatalf("ERROR: Problem opening config file: %s", err)
		}
		defer f.Close()

		got := test.cfg.ReplaceConfigFromReader(test.key, test.val, f)

		if !strings.Contains(got, test.commented) {
			t.Errorf("%s\ngot: %s\nMissing commented: %s", test.desc, got, test.commented)
		}

		if !strings.Contains(got, test.added) {
			t.Errorf("%s\ngot: %s\nMissing added: %s", test.desc, got, test.added)
		}
	}
}

func TestParseKeyValueFile(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting current dir: %s", err)
	}

	tests := []struct {
		desc       string
		configPath string
		lang       string
		want       ConfigFile
	}{
		{
			desc:       "(Python) Everything parses correctly",
			configPath: filepath.Join(dir, "testdata", "python_config"),
			lang:       "python",
			want: ConfigFile{
				Filepath:  filepath.Join(dir, "testdata"),
				Filename:  "python_config",
				Lang:      "python",
				OAuthType: InstalledApp,
				ConfigKeys: ConfigKeys{
					ClientID:         "0123456789-GoodClientID.apps.googleusercontent.com",
					ClientSecret:     "GoodClientSecret",
					DevToken:         "GoodDevToken",
					RefreshToken:     "1/PG1Ap6P-Good_Refresh_Token",
					DelegatedAccount: "example@some.web.site.com",
				},
			},
		},
		{
			desc:       "(Ruby) Parses with comments and non-supported config",
			configPath: filepath.Join(dir, "testdata", "ruby_config"),
			lang:       "ruby",
			want: ConfigFile{
				Filepath:  filepath.Join(dir, "testdata"),
				Filename:  "ruby_config",
				Lang:      "ruby",
				OAuthType: InstalledApp,
				ConfigKeys: ConfigKeys{
					ClientID: "GoodClientID",
				},
			},
		},
		{
			desc:       "(PHP) Can parse values with quotes",
			configPath: filepath.Join(dir, "testdata", "php_config"),
			lang:       "php",
			want: ConfigFile{
				Filepath:  filepath.Join(dir, "testdata"),
				Filename:  "php_config",
				Lang:      "php",
				OAuthType: InstalledApp,
				ConfigKeys: ConfigKeys{
					ClientID:     "GoodClientID",
					ClientSecret: "GoodClientSecret",
					DevToken:     "GoodDevToken",
					RefreshToken: "GoodRefreshToken",
				},
			},
		},
		{
			desc:       "(Java) Everything parses correctly",
			configPath: filepath.Join(dir, "testdata", "java_config"),
			lang:       "java",
			want: ConfigFile{
				Filepath:  filepath.Join(dir, "testdata"),
				Filename:  "java_config",
				Lang:      "java",
				OAuthType: InstalledApp,
				ConfigKeys: ConfigKeys{
					ClientID:     "GoodClientID",
					ClientSecret: "GoodClientSecret",
					DevToken:     "GoodDevToken",
					RefreshToken: "GoodRefreshToken",
				},
			},
		},
	}

	for _, test := range tests {
		got, err := ParseKeyValueFile(test.lang, test.configPath, InstalledApp)

		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("ParseKeyValueFile(%s, %s):\nTest Case: %s\nReturned diff (-want -> +got):\n%s",
				test.lang, test.configPath, test.desc, diff)
		}

		if err != nil {
			t.Errorf("[%s] Error: %s", test.desc, errstring(err))
		}
	}
}

func TestParseXMLFile(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting current dir: %s", err)
	}

	tests := []struct {
		desc       string
		configPath string
		lang       string
		want       ConfigFile
		errstr     string
	}{
		{
			desc:       "(.NET) Everything parses correctly",
			configPath: filepath.Join(dir, "testdata", "dotnet_config1"),
			lang:       "dotnet",
			want: ConfigFile{
				Filepath:  filepath.Join(dir, "testdata"),
				Filename:  "dotnet_config1",
				Lang:      "dotnet",
				OAuthType: InstalledApp,
				ConfigKeys: ConfigKeys{
					ClientID:         "0123456789-GoodClientID.apps.googleusercontent.com",
					ClientSecret:     "GoodClientSecret",
					DevToken:         "GoodDevToken",
					RefreshToken:     "1/PG1Ap6P-Good_Refresh_Token",
					PrivateKeyPath:   "GoodPath",
					DelegatedAccount: "example@some.website.com",
				},
			},
		},
		{
			desc:       "(.NET) Malformed XML",
			configPath: filepath.Join(dir, "testdata", "dotnet_config2"),
			lang:       "dotnet",
			want: ConfigFile{
				Filepath:  filepath.Join(dir, "testdata"),
				Filename:  "dotnet_config2",
				Lang:      "dotnet",
				OAuthType: InstalledApp,
			},
			errstr: "XML syntax error",
		},
	}

	for _, test := range tests {
		got, err := ParseXMLFile(test.configPath, InstalledApp)

		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("ParseXMLFile(%s):\nTest Case: %s\nReturned diff (-want -> +got):\n%s",
				test.configPath, test.desc, diff)
		}

		if err != nil && !strings.Contains(err.Error(), test.errstr) {
			t.Errorf("%s\nParseXMLFile(%s):\nError: %s", test.desc, test.configPath, errstring(err))
		}
	}
}

func TestParseServiceAccJSON(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting current dir: %s", err)
	}

	tests := []struct {
		desc   string
		c      ConfigFile
		want   ServiceAccountInfo
		errstr string
	}{
		{
			desc: "Successfully parses service account JSON",
			c: ConfigFile{
				ConfigKeys: ConfigKeys{
					PrivateKeyPath: filepath.Join(dir, "testdata", "service_account.json"),
				},
			},
			want: ServiceAccountInfo{
				Type:                    "service_account",
				ProjectID:               "project-1234567",
				PrivateKeyID:            "00000iabcadfad",
				PrivateKey:              "-----BEGIN PRIVATE KEY-----\nabcdefg-----END PRIVATE KEY-----\n",
				ClientEmail:             "example@some.website.com",
				ClientID:                "11111",
				AuthURI:                 "https://accounts.google.com/o/oauth2/auth",
				TokenURI:                "https://oauth2.googleapis.com/token",
				AuthProviderX509CertURL: "https://www.googleapis.com/oauth2/v1/certs",
				ClientX509CertURL:       "https://www.googleapis.com/something",
			},
		},
		{
			desc: "Cannot read JSON file",
			c: ConfigFile{
				ConfigKeys: ConfigKeys{
					PrivateKeyPath: "/tmp/this/is/my/path",
				},
			},
			want:   ServiceAccountInfo{},
			errstr: "no such file or directory",
		},
	}

	for _, test := range tests {
		err := test.c.parseServiceAccJSON()

		if diff := pretty.Compare(test.want, test.c.ServiceAccountInfo); diff != "" {
			t.Errorf("parseServiceAccJSON():\nTest Case: %s\nReturned diff (-want -> +got):\n%s",
				test.desc, diff)
		}

		if err != nil && !strings.Contains(errstring(err), test.errstr) {
			t.Errorf("[%s] Error: %s", test.desc, errstring(err))
		}
	}
}

func TestCheckGoVersion(t *testing.T) {
	tests := []struct {
		desc    string
		version string
		want    error
	}{
		{
			desc:    "Version go1.11 is supported",
			version: "go1.11",
			want:    nil,
		},
		{
			desc:    "Version go2.0 is supported",
			version: "go2.0",
			want:    nil,
		},
		{
			desc:    "Version go1.12.9 is supported",
			version: "go1.12.9",
			want:    nil,
		},
		{
			desc:    "Version go1.13rc1 is supported",
			version: "go1.13rc1",
			want:    nil,
		},
		{
			desc:    "Version 1.12 is supported",
			version: "1.12",
			want:    nil,
		},
		{
			desc:    "Version go1.9 is not supported",
			version: "go1.9",
			want:    fmt.Errorf("minimum required"),
		},
		{
			desc:    "Version go#&^% is not supported",
			version: "go#&^%",
			want:    fmt.Errorf("too short"),
		},
		{
			desc:    "Version go1.rc is not supported",
			version: "go1.rc",
			want:    fmt.Errorf("could not parse"),
		},
		{
			desc:    "Version a.b.c is not supported",
			version: "a.b.c",
			want:    fmt.Errorf("could not parse"),
		},
	}

	for _, test := range tests {
		got := checkGoVersion(test.version)

		if !strings.Contains(errstring(got), errstring(test.want)) {
			t.Errorf("[%s] want: %v, got: %s", test.desc, test.want, got)
		}
	}
}

func errstring(err error) string {
	if err != nil {
		return err.Error()
	}
	return "nil"
}
