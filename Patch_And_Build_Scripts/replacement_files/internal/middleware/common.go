package middleware

import (
	"fmt"
	loginmodel "honoka-chan/internal/model/login"
	usermodel "honoka-chan/internal/model/user"
	"honoka-chan/internal/session"
	honokautils "honoka-chan/internal/utils"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const legacyAuthErrorBody = `{"code":20001,"message":""}`

func abortLegacyAuth(ctx *gin.Context, ss *session.Session, reason string) {
	log.Printf("authentication rejected for %s: %s", ctx.Request.URL.Path, reason)
	ss.Rollback()
	ctx.String(http.StatusForbidden, legacyAuthErrorBody)
	ctx.Abort()
}

// adoptLegacyAuthorizeToken is a controlled fallback for an existing local
// account whose old Termux token was not present in the new auth_key table.
// It is only allowed while that account is marked force_relogin, and the flag
// is cleared immediately after the one-time adoption.
func adoptLegacyAuthorizeToken(ss *session.Session, token string, userID int) (bool, error) {
	if token == "" || userID <= 0 {
		return false, nil
	}

	pref := usermodel.UserPref{}
	hasPref, err := ss.UserEng.Table(new(usermodel.UserPref)).
		Where("user_id = ?", userID).
		Get(&pref)
	if err != nil || !hasPref || !pref.ForceRelogin {
		return false, err
	}

	userExists, err := ss.UserEng.Table(new(usermodel.Users)).
		Where("user_id = ?", userID).
		Exist()
	if err != nil || !userExists {
		return false, err
	}

	_, err = ss.UserEng.Insert(&loginmodel.AuthKey{
		AuthorizeToken: token,
		UserID:         userID,
		InsertDate:     time.Now().Format("2006-01-02 15:04:05"),
	})
	if err != nil {
		return false, err
	}
	if err := usermodel.ClearUserForceRelogin(ss.UserEng, userID); err != nil {
		return false, err
	}

	log.Printf("adopted legacy authorize token for user %d", userID)
	return true, nil
}

func Common(ctx *gin.Context) {
	reqData := ""
	if form, err := ctx.MultipartForm(); err == nil {
		if v, ok := form.Value["request_data"]; ok && len(v) > 0 {
			reqData = v[0]
		}
	}
	if reqData == "" {
		reqData = ctx.PostForm("request_data")
	}
	ctx.Set("request_data", reqData)

	uid := ctx.GetHeader("User-ID")

	ss := session.Attach(ctx)
	ctx.Set("session", ss)
	defer ss.FinalizeOrRollback()

	if ctx.IsAborted() {
		return
	}

	authorize := ctx.GetHeader("Authorize")
	params, err := url.ParseQuery(authorize)
	if err != nil {
		abortLegacyAuth(ctx, ss, "malformed Authorize header")
		return
	}

	nonce, _ := strconv.Atoi(params.Get("nonce"))
	nonce++
	ctx.Set("nonce", nonce)

	token := params.Get("token")
	ctx.Set("token", token)

	if !honokautils.IsMainLoginEndpoint(ctx.Request.URL.Path) {
		userID, err := strconv.Atoi(uid)
		if err != nil || userID <= 0 {
			abortLegacyAuth(ctx, ss, "missing or invalid User-ID")
			return
		}

		valid, err := ss.IsAuthorizeTokenForUser(token, userID)
		if ss.CheckErr(err) {
			return
		}
		if !valid {
			adopted, err := adoptLegacyAuthorizeToken(ss, token, userID)
			if ss.CheckErr(err) {
				return
			}
			if !adopted {
				abortLegacyAuth(ctx, ss, "authorize token is absent or stale")
				return
			}
		}

		ctx.Set("userid", uid)
		ss.LoadUser(uid)
		if ss.Done() {
			return
		}

		// The Android wrapper keeps the local client's existing token alive.
		// Mainline marks all users for relogin at startup and after some WebUI
		// edits, but the legacy client treats that rejection as maintenance.
		// Clear the flag and continue with the already validated token.
		if ss.UserPref.ForceRelogin {
			if err := usermodel.ClearUserForceRelogin(ss.UserEng, userID); ss.CheckErr(err) {
				return
			}
			ss.UserPref.ForceRelogin = false
			log.Printf("cleared force_relogin for active user %d", userID)
		}
	}

	ctx.Header("user_id", uid)
	ctx.Header("authorize", fmt.Sprintf("consumerKey=lovelive_test&timeStamp=%d&version=1.1&token=%s&nonce=%d&user_id=%s&requestTimeStamp=%d", time.Now().Unix(), token, nonce, uid, time.Now().Unix()))

	ctx.Header("Content-Type", "application/json; charset=utf-8")
	ctx.Header("X-Powered-By", "KLab Native APP Platform")
	ctx.Header("server_version", "20120129")
	ctx.Header("Server-Version", "97.4.6")
	ctx.Header("version_up", "0")
	ctx.Header("status_code", strconv.Itoa(http.StatusOK))

	ctx.Next()

	if ctx.Writer.Header().Get("Maintenance") == "1" {
		log.Printf("maintenance response: %s %s status=%d", ctx.Request.Method, ctx.Request.URL.Path, ctx.Writer.Status())
	}
}
