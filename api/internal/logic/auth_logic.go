package logic

import (
	"context"
	"errors"
	"strings"

	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"
	"ai-gozero-agent/api/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AuthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AuthLogic {
	return &AuthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AuthLogic) Register(req *types.RegisterReq) (*types.LoginResp, error) {
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" || strings.TrimSpace(req.StudentNo) == "" {
		return nil, errors.New("username/password/studentNo required")
	}

	if _, err := l.svcCtx.UserModel.FindByUsername(l.ctx, req.Username); err == nil {
		return nil, errors.New("username already exists")
	} else if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	userId, err := l.svcCtx.UserModel.Insert(l.ctx, &model.User{
		Username:     req.Username,
		PasswordHash: hash,
		Role:         "student",
		RealName:     strings.TrimSpace(req.RealName),
		Status:       1,
	})
	if err != nil {
		return nil, err
	}

	studentId, err := l.svcCtx.StudentModel.Insert(l.ctx, &model.Student{
		UserID:    userId,
		StudentNo: strings.TrimSpace(req.StudentNo),
		Skills:    "[]",
	})
	if err != nil {
		return nil, err
	}

	token, expireAt, err := utils.GenerateJWTToken(
		l.svcCtx.Config.Auth.AccessSecret,
		l.svcCtx.Config.Auth.AccessExpire,
		userId,
		"student",
	)
	if err != nil {
		return nil, err
	}

	return &types.LoginResp{
		Token:    token,
		ExpireAt: expireAt,
		User: types.UserProfile{
			UserId:    userId,
			Username:  req.Username,
			Role:      "student",
			StudentId: studentId,
			RealName:  strings.TrimSpace(req.RealName),
			Status:    1,
		},
	}, nil
}

func (l *AuthLogic) Login(req *types.LoginReq) (*types.LoginResp, error) {
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		return nil, errors.New("username/password required")
	}

	u, err := l.svcCtx.UserModel.FindByUsername(l.ctx, req.Username)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, errors.New("invalid username or password")
		}
		return nil, err
	}

	if u.Status != 1 {
		return nil, errors.New("user disabled")
	}

	if !utils.VerifyPassword(u.PasswordHash, req.Password) {
		return nil, errors.New("invalid username or password")
	}

	var studentId int64
	if u.Role == "student" {
		s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, u.ID)
		if err != nil {
			return nil, err
		}
		studentId = s.ID
	}

	token, expireAt, err := utils.GenerateJWTToken(
		l.svcCtx.Config.Auth.AccessSecret,
		l.svcCtx.Config.Auth.AccessExpire,
		u.ID,
		u.Role,
	)
	if err != nil {
		return nil, err
	}

	return &types.LoginResp{
		Token:    token,
		ExpireAt: expireAt,
		User: types.UserProfile{
			UserId:    u.ID,
			Username:  u.Username,
			Role:      u.Role,
			StudentId: studentId,
			RealName:  u.RealName,
			Phone:     u.Phone,
			Email:     u.Email,
			Status:    u.Status,
		},
	}, nil
}

func (l *AuthLogic) Profile(userId int64) (*types.UserProfile, error) {
	u, err := l.svcCtx.UserModel.FindById(l.ctx, userId)
	if err != nil {
		return nil, err
	}

	var studentId int64
	if u.Role == "student" {
		s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, u.ID)
		if err == nil {
			studentId = s.ID
		}
	}

	return &types.UserProfile{
		UserId:    u.ID,
		Username:  u.Username,
		Role:      u.Role,
		StudentId: studentId,
		RealName:  u.RealName,
		Phone:     u.Phone,
		Email:     u.Email,
		Status:    u.Status,
	}, nil
}

func (l *AuthLogic) ChangePassword(userId int64, req *types.ChangePasswordReq) error {
	if req.OldPassword == "" || req.NewPassword == "" {
		return errors.New("oldPassword/newPassword required")
	}

	u, err := l.svcCtx.UserModel.FindById(l.ctx, userId)
	if err != nil {
		return err
	}

	if !utils.VerifyPassword(u.PasswordHash, req.OldPassword) {
		return errors.New("old password incorrect")
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	return l.svcCtx.UserModel.UpdatePassword(l.ctx, userId, hash)
}
