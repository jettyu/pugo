package service

import (
	"errors"
	"pugo/src/core"
	"pugo/src/model"
	"pugo/src/utils"
	"strings"
)

var (
	Comment = new(CommentService)

	ErrCommentOriginMissing   = errors.New("comment-origin-missing")
	ErrCommentReferFail       = errors.New("comment-refer-error")
	ErrCommentContentTooShort = errors.New("comment-content-too-short")
	ErrCommentContentTooLong  = errors.New("comment-content-too-long")
	ErrCommentContentHref     = errors.New("comment-content-contains-href")
)

type CommentService struct{}

type CommentCreateOption struct {
	Name     string
	Email    string
	Url      string
	Content  string
	ParentId int64
	UserId   int64
	Type     string
	Id       int64
}

func (cs *CommentService) Create(v interface{}) (*Result, error) {
	opt, ok := v.(CommentCreateOption)
	if !ok {
		return nil, ErrServiceFuncNeedType(cs.Create, opt)
	}
	c := &model.Comment{
		Name:      opt.Name,
		UserId:    opt.UserId,
		Email:     opt.Email,
		Url:       opt.Url,
		AvatarUrl: utils.Gravatar(opt.Email),
		Body:      opt.Content,
		Status:    model.COMMENT_STATUS_WAIT,
		//From     int   `xorm:"INT(8) index(from)" json:"-"`
		FromId:   opt.Id,
		ParentId: opt.ParentId,
	}

	// filter content
	if len(c.Body) < Setting.Comment.MinLength {
		return nil, ErrCommentContentTooShort
	}
	if len(c.Body) > Setting.Comment.MaxLength {
		return nil, ErrCommentContentTooLong
	}
	if strings.Contains(c.Body, "href=") {
		return nil, ErrCommentContentHref
	}

	// set origin
	if opt.Type == "article" {
		c.From = model.COMMENT_FROM_ARTICLE
	}
	if opt.Type == "page" {
		c.From = model.COMMENT_FROM_PAGE
	}
	if c.From == 0 {
		return nil, ErrCommentOriginMissing
	}

	// check refer
	if Setting.Comment.CheckRefer {
		// todo : check refer
	}

	// try to read user
	if opt.UserId == 0 && opt.Email != "" {
		if user, _ := User.getUserBy("email", opt.Email); user != nil && user.Id > 0 {
			opt.UserId = user.Id
		}
	}

	// update status
	if Setting.Comment.CheckAll {
		// check all comment
		return newResult(cs.Create, c), nil
	}
	if Setting.Comment.CheckNoPass {
		count, err := cs.countEmailComment(c.Email)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			c.Status = model.COMMENT_STATUS_APPROVED
		}
	}

	return newResult(cs.Create, c), nil
}

func (cs *CommentService) countEmailComment(email string) (int64, error) {
	return core.Db.Where("email = ? AND status = ?", email, model.COMMENT_STATUS_APPROVED).Count(new(model.Comment))
}

func (cs *CommentService) Save(v interface{}) (*Result, error) {
	c, ok := v.(*model.Comment)
	if !ok {
		return nil, ErrServiceFuncNeedType(cs.Save, c)
	}
	// save comment
	if _, err := core.Db.Insert(c); err != nil {
		return nil, err
	}

	// update count
	if c.IsTopApproved() {
		if err := cs.updateCommentCount(c.From, c.FromId); err != nil {
			return nil, err
		}
	}
	return newResult(cs.Save, c), nil
}

func (cs *CommentService) updateCommentCount(from int, id int64) error {
	count, err := core.Db.Where("from = ? AND from_id = ? AND status = ? AND parent_id = ?", from, id, model.COMMENT_STATUS_APPROVED, 0).Count(new(model.Comment))
	if err != nil {
		return err
	}
	if from == model.COMMENT_FROM_ARTICLE {
		if _, err := core.Db.Exec("UPDATE article SET comments = ? WHERE id = ?", count, id); err != nil {
			return err
		}
	}
	if from == model.COMMENT_FROM_PAGE {
		if _, err := core.Db.Exec("UPDATE page SET comments = ? WHERE id = ?", count, id); err != nil {
			return err
		}
	}
	return nil
}

type CommentListOption struct {
	From          int
	FromId        int64
	Page          int
	Size          int
	Order         string
	Status        int
	AllApproved   bool
	AllAccessible bool
}

func prepareCommentListOption(opt CommentListOption) CommentListOption {
	if opt.Size == 0 {
		opt.Size = 1000
	}
	if opt.Page < 1 {
		opt.Page = 1
	}
	if opt.Order == "" {
		opt.Order = "create_time DESC"
	}
	// set default status
	if opt.Status == 0 {
		if !opt.AllApproved && !opt.AllApproved {
			opt.AllApproved = true
			return opt
		}
		if opt.AllApproved {
			// use status to instead
			opt.Status = model.COMMENT_STATUS_APPROVED
		}
	}
	return opt
}

func (cs *CommentService) List(v interface{}) (*Result, error) {
	opt, ok := v.(CommentListOption)
	if !ok {
		return nil, ErrServiceFuncNeedType(cs.List, opt)
	}
	sess := core.Db.NewSession().Limit(opt.Size, (opt.Page-1)*opt.Size).OrderBy(opt.Order)
	defer sess.Close()
	if opt.Status > 0 {
		sess.Where("status = ?", opt.Status)
	} else {
		if opt.AllApproved {
			sess.Where("status = ?", model.COMMENT_STATUS_APPROVED)
		}
		if opt.AllAccessible {
			sess.Where("status < ?", model.COMMENT_STATUS_SPAM)
		}
	}
	comments := make([]*model.Comment, 0)
	if err := sess.Find(&comments); err != nil {
		return nil, err
	}
	return newResult(cs.List, &comments), nil
}
