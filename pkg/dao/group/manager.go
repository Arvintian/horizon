package group

import (
	"context"
	"errors"

	"g.hz.netease.com/horizon/pkg/lib/q"
)

var (
	// Mgr is the global group manager
	Mgr = New()

	// ErrHasChildren used when delete a group which still has some children
	ErrHasChildren = errors.New("children exist, cannot be deleted")
)

const (
	// _updateAt one of the field of the group table
	_updateAt = "updated_at"

	// _parentID one of the field of the group table
	_parentID = "parent_id"
)

type Manager interface {
	// Create a group
	Create(ctx context.Context, group *Group) (uint, error)
	// Delete a group by id
	Delete(ctx context.Context, id uint) (int64, error)
	// GetByID get a group by id
	GetByID(ctx context.Context, id uint) (*Group, error)
	// GetByIDs get groups by ids
	GetByIDs(ctx context.Context, ids []uint) ([]*Group, error)
	// GetByPaths get groups by paths
	GetByPaths(ctx context.Context, paths []string) ([]*Group, error)
	// GetByNameFuzzily get groups that fuzzily matching the given name
	GetByNameFuzzily(ctx context.Context, name string) ([]*Group, error)
	// UpdateBasic update basic info of a group
	UpdateBasic(ctx context.Context, group *Group) error
	// GetSubGroupsUnderParentIDs get subgroups under the given parent groups without paging
	GetSubGroupsUnderParentIDs(ctx context.Context, parentIDs []uint) ([]*Group, error)
	// Transfer move a group under another parent group
	Transfer(ctx context.Context, id, newParentID uint) error
	// GetSubGroups get subgroups of a parent group, order by updateTime desc by default with paging
	GetSubGroups(ctx context.Context, id uint, pageNumber, pageSize int) ([]*Group, int64, error)
}

type manager struct {
	dao DAO
}

func (m manager) GetSubGroups(ctx context.Context, id uint, pageNumber, pageSize int) ([]*Group, int64, error) {
	query := formatListGroupQuery(id, pageNumber, pageSize)
	return m.dao.List(ctx, query)
}

func New() Manager {
	return &manager{dao: newDAO()}
}

func (m manager) Transfer(ctx context.Context, id, newParentID uint) error {
	return m.dao.Transfer(ctx, id, newParentID)
}

func (m manager) GetByPaths(ctx context.Context, paths []string) ([]*Group, error) {
	return m.dao.GetByPaths(ctx, paths)
}

func (m manager) GetByIDs(ctx context.Context, ids []uint) ([]*Group, error) {
	return m.dao.GetByIDs(ctx, ids)
}

func (m manager) GetByNameFuzzily(ctx context.Context, name string) ([]*Group, error) {
	return m.dao.GetByNameFuzzily(ctx, name)
}

func (m manager) Create(ctx context.Context, group *Group) (uint, error) {
	id, err := m.dao.Create(ctx, group)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (m manager) Delete(ctx context.Context, id uint) (int64, error) {
	count, err := m.dao.CountByParentID(ctx, id)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, ErrHasChildren
	}
	// todo check application children exist

	return m.dao.Delete(ctx, id)
}

func (m manager) GetByID(ctx context.Context, id uint) (*Group, error) {
	return m.dao.GetByID(ctx, id)
}

func (m manager) UpdateBasic(ctx context.Context, group *Group) error {
	// check record exist
	_, err := m.dao.GetByID(ctx, group.ID)
	if err != nil {
		return err
	}

	// check if there's record with the same parentID and name
	err = m.dao.CheckNameUnique(ctx, group)
	if err != nil {
		return err
	}
	// check if there's a record with the same parentID and path
	err = m.dao.CheckPathUnique(ctx, group)
	if err != nil {
		return err
	}

	return m.dao.UpdateBasic(ctx, group)
}

func (m manager) GetSubGroupsUnderParentIDs(ctx context.Context, parentIDs []uint) ([]*Group, error) {
	query := q.New(q.KeyWords{
		_parentID: parentIDs,
	})
	return m.dao.ListWithoutPage(ctx, query)
}

// formatListGroupQuery query info for listing groups under a parent group, order by updated_at desc by default
func formatListGroupQuery(id uint, pageNumber, pageSize int) *q.Query {
	query := q.New(q.KeyWords{
		_parentID: id,
	})
	query.PageNumber = pageNumber
	query.PageSize = pageSize
	// sort by updated_at desc default，let newer items be in head
	s := q.NewSort(_updateAt, true)
	query.Sorts = []*q.Sort{s}

	return query
}
