/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package translation

import (
	"aicsd/pkg"
	"aicsd/pkg/werrors"
	"encoding/json"
	"fmt"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// NewBundler creates an i18n.Bundle for service internationalization
// with the default language of English specified.
func NewBundler(localizationFiles []string) (*i18n.Bundle, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	for _, locFile := range localizationFiles {
		_, err := bundle.LoadMessageFile(locFile)
		if err != nil {
			return nil, werrors.WrapMsgf(err, "failed to LoadMessageFile")
		}
	}
	return bundle, nil
}

// TranslateField translates a struct field to the appropriate localization.
func TranslateField(loc *i18n.Localizer, field string) (string, error) {
	return loc.Localize(&i18n.LocalizeConfig{
		MessageID: field})
}

// TranslateErrorDetails translates the error details for jobs and files.
// This function expects both the pkg.UserFacingError.Error and pkg.UserFacingError.Owner fields to contain values,
// and will determine that translations are not needed if one or both are empty strings.
func TranslateErrorDetails(loc *i18n.Localizer, field *pkg.UserFacingError) (*pkg.UserFacingError, error) {
	// if fields empty, don't need to translate
	if field.Owner == "" || field.Error == "" {
		return field, nil // return no error for cases of jobs/files with no ErrorDetails to translate
	}

	// translate owner field, and notify err translating and give owner field back in English
	ownerTranslated, err := TranslateField(loc, field.Owner)
	if err != nil {
		notifyFailureUponTranslationOwner, err2 := TranslateField(loc, pkg.GetErrorType(pkg.ErrTranslating.Error()))
		if err2 != nil {
			return nil, err2
		}

		return pkg.CreateUserFacingError(field.Owner, fmt.Errorf(notifyFailureUponTranslationOwner)), pkg.ErrTranslating
	}

	// translate err message field, and upon err set field to ErrTranslating + field.Error in English
	errMsgTranslated, err := TranslateField(loc, pkg.GetErrorType(field.Error))
	if err != nil {
		notifyFailureUponTranslationError, err2 := TranslateField(loc, pkg.GetErrorType(pkg.ErrTranslating.Error()))
		if err2 != nil {
			return nil, err2
		}

		return pkg.CreateUserFacingError(ownerTranslated, fmt.Errorf(notifyFailureUponTranslationError, field.Error)), pkg.ErrTranslating
	}

	// return translated fields
	return pkg.CreateUserFacingError(ownerTranslated, fmt.Errorf(errMsgTranslated)), nil
}
