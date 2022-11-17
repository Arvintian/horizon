package token

import (
	"g.hz.netease.com/horizon/core/common"
	"g.hz.netease.com/horizon/core/controller/oauthcheck"
	herrors "g.hz.netease.com/horizon/core/errors"
	"g.hz.netease.com/horizon/pkg/auth"
	perror "g.hz.netease.com/horizon/pkg/errors"
	"g.hz.netease.com/horizon/pkg/server/middleware"
	"g.hz.netease.com/horizon/pkg/server/response"
	"g.hz.netease.com/horizon/pkg/util/log"
	"g.hz.netease.com/horizon/pkg/util/sets"
	"github.com/gin-gonic/gin"
)

var RequestInfoFty auth.RequestInfoFactory

func init() {
	RequestInfoFty = auth.RequestInfoFactory{
		APIPrefixes: sets.NewString("apis"),
	}
}

const (
	CheckResult = "Result"
)

func MiddleWare(oauthCtl oauthcheck.Controller, skipMatchers ...middleware.Skipper) gin.HandlerFunc {
	return middleware.New(func(c *gin.Context) {
		// 1. get user from token and set user context
		token, err := common.GetToken(c)
		if err != nil {
			log.Warning(c, "Have not got token")
			c.Next()
			return
		}

		// 2. check token valid
		if err := oauthCtl.ValidateToken(c, token); err != nil {
			if perror.Cause(err) == herrors.ErrOAuthAccessTokenExpired {
				response.AbortWithUnauthorized(c, common.CodeExpired, err.Error())
				return
			}
			if e, ok := perror.Cause(err).(*herrors.HorizonErrNotFound); ok {
				response.AbortWithUnauthorized(c, common.Unauthorized, e.Error())
				return
			}
			response.AbortWithUnauthorized(c, common.InternalError, err.Error())
			return
		}

		// 3. do scope check(get requestInfo, and do check)
		user, err := oauthCtl.LoadAccessTokenUser(c, token)
		if err != nil {
			if e, ok := perror.Cause(err).(*herrors.HorizonErrNotFound); ok {
				response.AbortWithUnauthorized(c, common.Unauthorized, e.Error())
				return
			}
			response.AbortWithInternalError(c, err.Error())
			return
		}
		common.SetUser(c, user)

		requestInfo, err := RequestInfoFty.NewRequestInfo(c.Request)
		if err != nil {
			response.AbortWithRequestError(c, common.RequestInfoError, err.Error())
			return
		}
		result, reason, err := oauthCtl.CheckScopePermission(c, token, *requestInfo)
		if err != nil {
			if e, ok := perror.Cause(err).(*herrors.HorizonErrNotFound); ok {
				response.AbortWithUnauthorized(c, common.Unauthorized, e.Error())
				return
			}
		}
		if !result {
			log.WithFiled(c, CheckResult, result).Warningf("reason = %s", reason)
			response.AbortWithForbiddenError(c, common.Forbidden, "")
			return
		}
		log.WithFiled(c, CheckResult, result).Infof("reason = %s", reason)
		c.Next()
	}, skipMatchers...)
}