package types

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterReq struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	StudentNo string `json:"studentNo"`
	RealName  string `json:"realName,optional"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type UserProfile struct {
	UserId    int64  `json:"userId"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	StudentId int64  `json:"studentId,omitempty"`
	RealName  string `json:"realName"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Status    int16  `json:"status"`
}

type LoginResp struct {
	Token    string      `json:"token"`
	ExpireAt int64       `json:"expireAt"`
	User     UserProfile `json:"user"`
}
