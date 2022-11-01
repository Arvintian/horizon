package idp

import (
	"net/http"
	"strconv"

	"g.hz.netease.com/horizon/core/common"
	"g.hz.netease.com/horizon/core/controller/idp"
	herrors "g.hz.netease.com/horizon/core/errors"
	userauth "g.hz.netease.com/horizon/pkg/authentication/user"
	perror "g.hz.netease.com/horizon/pkg/errors"
	"g.hz.netease.com/horizon/pkg/server/response"
	"g.hz.netease.com/horizon/pkg/server/rpcerror"
	usermodel "g.hz.netease.com/horizon/pkg/user/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

// for path variable
var (
	_idp = "idp"
)

// for query
var (
	_oidcCode    = "code"
	_oidcState   = "state"
	_redirectURL = "redirectUrl"
)

type API struct {
	idpCtrl idp.Controller
	store   sessions.Store
}

func NewAPI(idpCtrl idp.Controller, store sessions.Store) *API {
	return &API{
		idpCtrl: idpCtrl,
		store:   store,
	}
}

func (a *API) ListAuthEndpoints(c *gin.Context) {
	redirectURL := c.Query(_redirectURL)
	endpoints, err := a.idpCtrl.ListAuthEndpoints(c, redirectURL)
	if err != nil {
		response.AbortWithRPCError(c, rpcerror.InternalError.WithErrMsg(err.Error()))
		return
	}
	c.PureJSON(http.StatusOK, response.NewResponseWithData(endpoints))
}

func (a *API) ListIDPs(c *gin.Context) {
	idps, err := a.idpCtrl.List(c)
	if err != nil {
		response.AbortWithRPCError(c, rpcerror.InternalError.WithErrMsg(err.Error()))
		return
	}
	response.SuccessWithData(c, idps)
}

func (a *API) LoginCallback(c *gin.Context) {
	code := c.Query(_oidcCode)
	state := c.Query(_oidcState)
	if code == "" || state == "" {
		response.AbortWithRPCError(c,
			rpcerror.ParamError.WithErrMsgf(
				"code and state should not be empty:\n"+
					"code = %s\n state = %s", code, state))
		return
	}

	var (
		user *usermodel.User
		err  error
	)

	if user, err = a.idpCtrl.Login(c, code, state); err != nil {
		response.AbortWithRPCError(c,
			rpcerror.InternalError.WithErrMsg(err.Error()))
		return
	}

	session := a.getSession(c)
	if session == nil {
		return
	}

	session.Values[common.SessionKeyAuthUser] = &userauth.DefaultInfo{
		Name:     user.Name,
		FullName: user.FullName,
		ID:       user.ID,
		Email:    user.Email,
		Admin:    user.Admin,
	}

	if err := session.Save(c.Request, c.Writer); err != nil {
		response.AbortWithRPCError(c,
			rpcerror.InternalError.WithErrMsgf(
				"saving session into backend or response failed:\n"+
					"err = %v", err))
		return
	}

	response.Success(c)
}

func (a *API) Logout(c *gin.Context) {
	session := a.getSession(c)
	if session == nil {
		return
	}

	session.Options.MaxAge = -1

	if err := session.Save(c.Request, c.Writer); err != nil {
		response.AbortWithRPCError(c,
			rpcerror.InternalError.WithErrMsgf(
				"saving session into backend or response failed:\n"+
					"err = %v", err))
		return
	}
}

func (a *API) CreateIDP(c *gin.Context) {
	createParam := idp.CreateIDPRequest{}
	err := c.ShouldBindJSON(&createParam)
	if err != nil {
		response.AbortWithRPCError(c,
			rpcerror.InternalError.WithErrMsgf(
				"binding creating params failed:\n"+
					"err = %v", err))
		return
	}

	_, err = a.idpCtrl.Create(c, &createParam)
	if err != nil {
		if _, ok := perror.Cause(err).(*herrors.HorizonErrCreateFailed); ok {
			response.AbortWithRPCError(
				c, rpcerror.ParamError.WithErrMsgf("failed to create idp\n"+
					"err = %v", err.Error()))
			return
		}
		response.AbortWithRPCError(
			c, rpcerror.InternalError.WithErrMsgf("failed to create idp\n"+
				"err = %v", err.Error()))
		return
	}
}

func (a *API) DeleteIDP(c *gin.Context) {
	idpIDStr := c.Param(_idp)

	var (
		idpID uint64
		err   error
	)

	if idpID, err = strconv.ParseUint(idpIDStr, 10, 64); err != nil {
		response.AbortWithRPCError(c, rpcerror.ParamError.WithErrMsg("idp ID is not found or invalid"))
		return
	}

	err = a.idpCtrl.Delete(c, uint(idpID))
	if err != nil {
		if _, ok := perror.Cause(err).(*herrors.HorizonErrNotFound); ok {
			response.AbortWithRPCError(
				c, rpcerror.NotFoundError.WithErrMsgf("idp with id = %d was not found", idpID),
			)
			return
		}
		response.AbortWithRPCError(c, rpcerror.InternalError.WithErrMsgf("failed to delete idp:\n"+
			"id = %d \nerr = %v", idpID, err))
		return
	}
}

func (a *API) UpdateIDP(c *gin.Context) {
	idpIDStr := c.Param(_idp)

	var (
		idpID uint64
		err   error
	)

	if idpID, err = strconv.ParseUint(idpIDStr, 10, 64); err != nil {
		response.AbortWithRPCError(c, rpcerror.ParamError.WithErrMsg("idp ID is not found or invalid"))
		return
	}

	updateParam := &idp.UpdateIDPRequest{}
	err = c.ShouldBindJSON(updateParam)
	if err != nil {
		response.AbortWithRPCError(c,
			rpcerror.InternalError.WithErrMsgf(
				"binding creating params failed:\n"+
					"err = %v", err))
		return
	}

	_, err = a.idpCtrl.Update(c, uint(idpID), updateParam)
	if err != nil {
		if _, ok := perror.Cause(err).(*herrors.HorizonErrNotFound); ok {
			response.AbortWithRPCError(
				c, rpcerror.NotFoundError.WithErrMsgf("idp with id = %d was not found", idpID),
			)
			return
		}
		response.AbortWithRPCError(c, rpcerror.InternalError.WithErrMsgf("failed to delete idp:\n"+
			"id = %d \nerr = %v", idpID, err))
		return
	}
}

func (a *API) GetByID(c *gin.Context) {
	idpIDStr := c.Param(_idp)

	var (
		idpID uint64
		err   error
	)

	if idpID, err = strconv.ParseUint(idpIDStr, 10, 64); err != nil {
		response.AbortWithRPCError(c, rpcerror.ParamError.WithErrMsg("idp ID is not found or invalid"))
		return
	}

	idp, err := a.idpCtrl.GetByID(c, uint(idpID))
	if err != nil {
		if _, ok := perror.Cause(err).(*herrors.HorizonErrNotFound); ok {
			response.AbortWithRPCError(
				c, rpcerror.NotFoundError.WithErrMsgf("idp with id = %d was not found", idpID),
			)
			return
		}
		response.AbortWithRPCError(c, rpcerror.InternalError.WithErrMsgf("failed to get idp:\n"+
			"id = %d \nerr = %v", idpID, err))
		return
	}
	response.SuccessWithData(c, idp)
}

func (a *API) GetDiscovery(ctx *gin.Context) {
	discovery := idp.Discovery{}
	err := ctx.ShouldBindJSON(&discovery)
	if err != nil {
		response.AbortWithRPCError(ctx,
			rpcerror.InternalError.WithErrMsgf(
				"binding creating params failed:\n"+
					"err = %v", err))
		return
	}

	config, err := a.idpCtrl.GetDiscovery(ctx, discovery)
	if err != nil {
		response.AbortWithRPCError(ctx,
			rpcerror.ParamError.WithErrMsgf(
				"failed to get discovery config\n"+
					"err = %v", err))
		return
	}
	response.SuccessWithData(ctx, config)
}

func (a *API) getSession(c *gin.Context) *sessions.Session {
	session, err := a.store.Get(c.Request, common.CookieKeyAuth)
	if err != nil {
		response.AbortWithRPCError(c,
			rpcerror.InternalError.WithErrMsgf(
				"session not found:\n"+
					"session name = %s\n err = %v", common.CookieKeyAuth, err))
		return nil
	}
	return session
}