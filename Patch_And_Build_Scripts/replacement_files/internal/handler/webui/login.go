package webui

import (
	"fmt"
	"honoka-chan/internal/middleware"
	"honoka-chan/internal/router"
	webuischema "honoka-chan/internal/schema/webui"
	"honoka-chan/pkg/db"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-think/openssl"
)

type webLoginAccount struct {
	UserID int    `xorm:"user_id"`
	Phone  string `xorm:"phone"`
}

func login(ctx *gin.Context) {
	user := strings.TrimSpace(ctx.PostForm("user"))
	pass := ctx.PostForm("pass")
	area := strings.TrimSpace(ctx.PostForm("area")) // 兼容旧版 WebUI 缓存页面。
	if user == "" || pass == "" {
		ctx.JSON(http.StatusOK, webuischema.Msg{
			Code:     1,
			Message:  "参数不完整！",
			Redirect: "",
		})
		return
	}

	userID, err := resolveWebLoginUser(user, area, openssl.Md5ToString(pass))
	if err != nil {
		ctx.JSON(http.StatusOK, webuischema.Msg{
			Code:     1,
			Message:  err.Error(),
			Redirect: "",
		})
		return
	}

	session := sessions.Default(ctx)
	session.Options(sessions.Options{
		Path:   "/",
		MaxAge: 3600 * 24,
	})
	session.Set("userid", userID)
	if err := session.Save(); err != nil {
		ctx.JSON(http.StatusOK, webuischema.Msg{
			Code:     1,
			Message:  "登录状态保存失败！",
			Redirect: "",
		})
		return
	}

	ctx.JSON(http.StatusOK, webuischema.Msg{
		Code:     0,
		Message:  "登录成功！",
		Redirect: "/admin/index",
	})
}

// resolveWebLoginUser preserves the current mainline login format while also
// accepting accounts created by the former Termux core.
//
// Current mainline stores phone as the normalized account name, for example:
//
//	13800000000
//
// The former Termux WebUI stored the same account as:
//
//	" 86-13800000000"
//
// Its login page submitted area, account and password separately. The current
// page no longer contains an area selector, so an exact lookup alone makes an
// upgraded database appear to have lost its WebUI account even though the row
// and password are still present.
func resolveWebLoginUser(user, area, passwordHash string) (int, error) {
	accounts := []webLoginAccount{}
	if err := db.UserEng.Table("users").
		Where("password = ?", passwordHash).
		Cols("user_id,phone").
		OrderBy("id ASC").
		Find(&accounts); err != nil {
		return 0, fmt.Errorf("读取账号失败！")
	}
	if len(accounts) == 0 {
		return 0, fmt.Errorf("账号不存在或者密码有误！")
	}

	input := strings.TrimSpace(user)
	exact := make([]webLoginAccount, 0, 1)
	legacy := make([]webLoginAccount, 0, 1)

	for _, account := range accounts {
		if account.UserID <= 0 {
			continue
		}
		stored := strings.TrimSpace(account.Phone)
		if stored == input {
			exact = append(exact, account)
			continue
		}

		// Cached old login pages may still post the area separately.
		if area != "" && stored == area+"-"+input {
			legacy = append(legacy, account)
			continue
		}

		// On the new login page a bare old account is accepted when the suffix
		// uniquely identifies one password-matching row. Entering the complete
		// "area-account" form remains available to disambiguate multiple rows.
		if dash := strings.Index(stored, "-"); dash >= 0 && dash+1 < len(stored) {
			storedArea := strings.TrimSpace(stored[:dash])
			storedUser := strings.TrimSpace(stored[dash+1:])
			if storedUser == input && (area == "" || storedArea == area) {
				legacy = append(legacy, account)
			}
		}
	}

	if len(exact) == 1 {
		return exact[0].UserID, nil
	}
	if len(exact) > 1 {
		return 0, fmt.Errorf("匹配到多个账号，请输入更完整的账号！")
	}
	if len(legacy) == 1 {
		return legacy[0].UserID, nil
	}
	if len(legacy) > 1 {
		return 0, fmt.Errorf("匹配到多个旧版账号，请输入完整账号，例如 86-账号！")
	}
	return 0, fmt.Errorf("账号不存在或者密码有误！")
}

func init() {
	router.AddHandler("admin", "GET", "/login", middleware.WebAuth, func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "admin/login.html", gin.H{})
	})
	router.AddHandler("admin", "POST", "/login", middleware.WebAuth, login)
}
