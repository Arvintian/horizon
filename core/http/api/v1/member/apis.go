package member

import (
	"fmt"
	"strconv"

	"g.hz.netease.com/horizon/core/common"
	"g.hz.netease.com/horizon/core/controller/member"
	membermodels "g.hz.netease.com/horizon/pkg/member/models"
	"g.hz.netease.com/horizon/pkg/server/response"
	"github.com/gin-gonic/gin"
)

const (
	_paramGroupID       = "groupID"
	_paramApplicationID = "applicationID"
	_paramMemberID      = "memberID"
)

type API struct {
	memberCtrl member.Controller
}

// NewAPI initializes a new group api
func NewAPI() *API {
	return &API{
		memberCtrl: member.Ctl,
	}
}

func (a *API) CreateGroupMember(c *gin.Context) {
	resourceIDStr := c.Param(_paramGroupID)
	uintID, err := strconv.ParseUint(resourceIDStr, 10, 0)
	if err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			fmt.Sprintf("%v", err))
		return
	}

	var postMember *member.PostMember
	err = c.ShouldBindJSON(&postMember)
	if err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestBody,
			fmt.Sprintf("%v", err))
		return
	}

	if uint(uintID) != postMember.ResourceID {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			"id not match")
		return
	}

	if err := validatePostMember(membermodels.TypeGroup, postMember); err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			err.Error())
		return
	}

	retMember, err := a.memberCtrl.CreateMember(c, postMember)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}
	response.SuccessWithData(c, retMember)
}

func (a *API) CreateApplicationMember(c *gin.Context) {
	resourceIDStr := c.Param(_paramApplicationID)
	uintID, err := strconv.ParseUint(resourceIDStr, 10, 0)
	if err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			fmt.Sprintf("%v", err))
		return
	}

	var postMember *member.PostMember
	err = c.ShouldBindJSON(&postMember)
	if err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestBody,
			fmt.Sprintf("%v", err))
		return
	}

	if uint(uintID) != postMember.ResourceID {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			"id not match")
		return
	}

	if err := validatePostMember(membermodels.TypeApplication, postMember); err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			err.Error())
		return
	}

	retMember, err := a.memberCtrl.CreateMember(c, postMember)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}
	response.SuccessWithData(c, retMember)
}

func (a *API) UpdateMember(c *gin.Context) {
	memberIDStr := c.Param(_paramMemberID)
	uintID, err := strconv.ParseUint(memberIDStr, 10, 0)
	if err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			fmt.Sprintf("memberid error,%v", err))
		return
	}
	var updateMember *member.UpdateMember
	if err = c.ShouldBindJSON(&updateMember); err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestBody,
			fmt.Sprintf("%v", err))
		return
	}

	if uint(uintID) != updateMember.ID {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			"id not match")
		return
	}

	if err := validRole(updateMember.Role); err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			err.Error())
		return
	}

	retMember, err := a.memberCtrl.UpdateMember(c, updateMember.ID, updateMember.Role)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}
	response.SuccessWithData(c, retMember)
}

func (a *API) DeleteMember(c *gin.Context) {
	memberIDStr := c.Param(_paramMemberID)
	uintID, err := strconv.ParseUint(memberIDStr, 10, 0)
	if err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			fmt.Sprintf("memberid error,%v", err))
		return
	}
	err = a.memberCtrl.RemoveMember(c, uint(uintID))
	if err != nil {
		response.AbortWithError(c, err)
		return
	}
	response.Success(c)
}

func (a *API) ListGroupMember(c *gin.Context) {
	resourceIDStr := c.Param(_paramGroupID)
	uintID, err := strconv.ParseUint(resourceIDStr, 10, 0)
	if err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			fmt.Sprintf("%v", err))
		return
	}

	members, err := a.memberCtrl.ListMember(c, membermodels.TypeGroupStr, uint(uintID))
	if err != nil {
		response.AbortWithError(c, err)
		return
	}
	membersResp := response.DataWithTotal{
		Items: members,
		Total: int64(len(members)),
	}
	response.SuccessWithData(c, membersResp)
}

func (a *API) ListApplicationMember(c *gin.Context) {
	resourceIDStr := c.Param(_paramApplicationID)
	uintID, err := strconv.ParseUint(resourceIDStr, 10, 0)
	if err != nil {
		response.AbortWithRequestError(c, common.InvalidRequestParam,
			fmt.Sprintf("%v", err))
		return
	}

	members, err := a.memberCtrl.ListMember(c, membermodels.TypeApplicationStr, uint(uintID))
	if err != nil {
		response.AbortWithError(c, err)
		return
	}
	membersResp := response.DataWithTotal{
		Items: members,
		Total: int64(len(members)),
	}
	response.SuccessWithData(c, membersResp)
}

// validatePostMember validate postMember body according to resourceType
func validatePostMember(resourceType membermodels.ResourceType, postMember *member.PostMember) error {
	if membermodels.ResourceType(postMember.ResourceType) != resourceType {
		return fmt.Errorf("resourceType not match")
	}
	if err := validResourceType(postMember.ResourceType); err != nil {
		return err
	}

	if err := validMemberType(postMember.MemberType); err != nil {
		return err
	}

	if err := validRole(postMember.Role); err != nil {
		return err
	}

	return nil
}

func validResourceType(resourceType string) error {
	switch membermodels.ResourceType(resourceType) {
	case membermodels.TypeGroup, membermodels.TypeApplication, membermodels.TypeApplicationCluster:
	default:
		return fmt.Errorf("invalid resourceType")
	}
	return nil
}

func validMemberType(memberType membermodels.MemberType) error {
	switch memberType {
	case membermodels.MemberUser:
	case membermodels.MemberGroup:
		return fmt.Errorf("this type of member is not supported yet")
	default:
		return fmt.Errorf("invalid memberType")
	}
	return nil
}

func validRole(role string) error {
	switch role {
	case membermodels.Owner, membermodels.Maintainer, membermodels.Developer, membermodels.Reporter, membermodels.Guest:
	default:
		return fmt.Errorf("invalid role")
	}
	return nil
}