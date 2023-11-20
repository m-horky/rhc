package l10n

import (
	"fmt"
	"github.com/snapcore/go-gettext"
)

var domain gettext.TextDomain
var locale gettext.Catalog

func init() {
	domain = gettext.TextDomain{Name: "rhc"}
	locale = domain.UserLocale()
}

// T localizes simple strings.
func T(str string, vars ...interface{}) string {
	translation := locale.Gettext(str)
	if len(vars) > 0 {
		translation = fmt.Sprintf(translation, vars)
	}
	return translation
}

// TN localizes strings with plurals.
func TN(singular, plural string, n uint32, vars ...interface{}) string {
	translation := locale.NGettext(singular, plural, n)
	if len(vars) > 0 {
		translation = fmt.Sprintf(translation, vars)
	}
	return translation
}

// TC localizes strings with contexts.
func TC(ctx, str string, vars ...interface{}) string {
	translation := locale.PGettext(ctx, str)
	if len(vars) > 0 {
		translation = fmt.Sprintf(translation, vars)
	}
	return translation
}

// TNC localizes strings with contexts and plurals.
func TNC(ctx, singular, plural string, n uint32, vars ...interface{}) string {
	translation := locale.NPGettext(ctx, singular, plural, n)
	if len(vars) > 0 {
		translation = fmt.Sprintf(translation, vars)
	}
	return translation
}
