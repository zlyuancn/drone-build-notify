/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/5/18
   Description :
-------------------------------------------------
*/

package template

import (
	"regexp"
	"strings"

	"github.com/zlyuancn/drone-build-notify/model"
)

var TemplateSyntaxRe = regexp.MustCompile(`{{ *?\w*? *?}}`)

func Render(text string, msg *model.Msg) string {
	return TemplateSyntaxRe.ReplaceAllStringFunc(text, func(s string) string {
		key := strings.TrimSpace(s[2 : len(s)-2])
		return msg.Get(key)
	})
}
