/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package translation_test

import (
	"aicsd/pkg"
	"aicsd/pkg/translation"
	"encoding/json"
	"fmt"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var (
	localizationFiles = []string{"../../pkg/translation/dictionary/en.json", "../../pkg/translation/dictionary/zh.json"}
)

func translationSetup(t *testing.T) (map[string]string, map[string]string, *i18n.Localizer) {
	t.Helper()
	// read in project dictionaries
	bytesEnglish, err := os.ReadFile(localizationFiles[0]) // read English dictionary
	require.NoError(t, err)
	require.NotEmpty(t, bytesEnglish)
	bytesChinese, err := os.ReadFile(localizationFiles[1]) // read Chinese dictionary
	require.NoError(t, err)
	require.NotEmpty(t, bytesChinese)

	var dictionaryEnglish, dictionaryChinese map[string]string
	require.NoError(t, json.Unmarshal(bytesEnglish, &dictionaryEnglish))
	require.NoError(t, json.Unmarshal(bytesChinese, &dictionaryChinese))

	testLocalizationBundle, err := translation.NewBundler(localizationFiles)
	require.NoError(t, err)
	testLocalizer := i18n.NewLocalizer(testLocalizationBundle, "zh") // TODO: switch to use chinese language const in pkg I made

	return dictionaryEnglish, dictionaryChinese, testLocalizer
}

func TestDictionaries(t *testing.T) {
	dictionaryEnglish, dictionaryChinese, _ := translationSetup(t)

	t.Run("happy path - verify keys are in both dictionaries", func(t *testing.T) {
		assert.Equal(t, len(dictionaryEnglish), len(dictionaryChinese))

		for englishKey, _ := range dictionaryEnglish {
			if _, ok := dictionaryChinese[englishKey]; !ok {
				t.Fatalf("Chinese dictionary is missing a translation for a value within the English dictionary for English key of %s", englishKey)
			}
		}
	})
}

func TestTranslateField(t *testing.T) {
	dictionaryEnglish, dictionaryChinese, testLocalizer := translationSetup(t)

	t.Run("verify English dictionary translates to Chinese dictionary", func(t *testing.T) {
		for k, _ := range dictionaryEnglish {
			got, err := translation.TranslateField(testLocalizer, k)
			assert.NoError(t, err)
			assert.Equal(t, got, dictionaryChinese[k])
		}
	})

	t.Run("edge cases - check strings not within dictionaries", func(t *testing.T) {
		for _, v := range []string{"", "fail", "testing"} {
			got, err := translation.TranslateField(testLocalizer, v)
			assert.Error(t, err)
			assert.Equal(t, got, "")
		}
	})
}

func TestTranslateErrorDetails(t *testing.T) {
	_, _, testLocalizer := translationSetup(t)

	tests := []struct {
		name                string // test name
		userFacingOwner     string
		userFacingErr       error
		wantUserFacingOwner func() string
		wantUserFacingError func() string
		wantFunctionError   error
	}{
		{
			name:            "edge case - no owner to translate, so no translations",
			userFacingOwner: "",
			userFacingErr:   pkg.ErrTranslating,
			wantUserFacingOwner: func() string {
				return ""
			},
			wantUserFacingError: func() string {
				return pkg.ErrTranslating.Error()
			},
			wantFunctionError: nil,
		},
		{
			name:            "edge case 2 - no owner to translate again, so no translations",
			userFacingOwner: "",
			userFacingErr:   pkg.ErrRetrieving,
			wantUserFacingOwner: func() string {
				return ""
			},
			wantUserFacingError: func() string {
				return pkg.ErrRetrieving.Error()
			},
			wantFunctionError: nil,
		},
		{
			name:            "edge case 3 - no error to translate, so no translations",
			userFacingOwner: pkg.OwnerNone,
			userFacingErr:   nil,
			wantUserFacingOwner: func() string {
				return pkg.OwnerNone
			},
			wantUserFacingError: func() string {
				return ""
			},
			wantFunctionError: nil,
		},
		{
			name:            "edge case 4 - no error to translate again, so no translations",
			userFacingOwner: pkg.OwnerFileWatcher,
			userFacingErr:   nil,
			wantUserFacingOwner: func() string {
				return pkg.OwnerFileWatcher
			},
			wantUserFacingError: func() string {
				return ""
			},
			wantFunctionError: nil,
		},
		{
			name:            "happy path - owner and error to translate",
			userFacingOwner: pkg.OwnerFileSenderGateway,
			userFacingErr:   pkg.ErrFileTransmitting,
			wantUserFacingOwner: func() string {
				translation, err := translation.TranslateField(testLocalizer, pkg.OwnerFileSenderGateway)
				require.NoError(t, err)
				return translation
			},
			wantUserFacingError: func() string {
				translation, err := translation.TranslateField(testLocalizer, pkg.GetErrorType(pkg.ErrFileTransmitting.Error()))
				require.NoError(t, err)
				return translation
			},
			wantFunctionError: nil,
		},
		{
			name:            "happy path - another owner and error to translate",
			userFacingOwner: pkg.OwnerTaskLauncher,
			userFacingErr:   pkg.ErrJobNoMatchingTask,
			wantUserFacingOwner: func() string {
				translation, err := translation.TranslateField(testLocalizer, pkg.OwnerTaskLauncher)
				require.NoError(t, err)
				return translation
			},
			wantUserFacingError: func() string {
				translation, err := translation.TranslateField(testLocalizer, pkg.GetErrorType(pkg.ErrJobNoMatchingTask.Error()))
				require.NoError(t, err)
				return translation
			},
			wantFunctionError: nil,
		},
		{
			name:            "invalid translation where owner is not within English dictionary",
			userFacingOwner: "invalidOwner",
			userFacingErr:   pkg.ErrTranslating,
			wantUserFacingOwner: func() string {
				return "invalidOwner"
			},
			wantUserFacingError: func() string {
				translation, err := translation.TranslateField(testLocalizer, pkg.GetErrorType(pkg.ErrTranslating.Error()))
				require.NoError(t, err)
				return translation
			},
			wantFunctionError: pkg.ErrTranslating,
		},
		{
			name:            "invalid translation where error msg is not within English dictionary",
			userFacingOwner: pkg.OwnerDataOrg,
			userFacingErr:   fmt.Errorf("some invalid error for testing"),
			wantUserFacingOwner: func() string {
				translation, err := translation.TranslateField(testLocalizer, pkg.OwnerDataOrg)
				require.NoError(t, err)
				return translation
			},
			wantUserFacingError: func() string {
				translation, err := translation.TranslateField(testLocalizer, pkg.GetErrorType(pkg.ErrTranslating.Error()))
				require.NoError(t, err)
				return translation
			},
			wantFunctionError: nil,
		},
		{
			name:            "invalid translation where owner and error msg is not within English dictionary",
			userFacingOwner: "invalidOwner",
			userFacingErr:   fmt.Errorf("some invalid error for testing"),
			wantUserFacingOwner: func() string {
				return "invalidOwner"
			},
			wantUserFacingError: func() string {
				translation, err := translation.TranslateField(testLocalizer, pkg.GetErrorType(pkg.ErrTranslating.Error()))
				require.NoError(t, err)
				return translation
			},
			wantFunctionError: pkg.ErrTranslating,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := translation.TranslateErrorDetails(testLocalizer, pkg.CreateUserFacingError(test.userFacingOwner, test.userFacingErr))
			assert.Equal(t, test.wantFunctionError, err)
			assert.Equal(t, test.wantUserFacingOwner(), got.Owner)
			assert.Equal(t, test.wantUserFacingError(), got.Error)
		})
	}
}
