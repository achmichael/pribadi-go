package http

type StandardResponse struct {
	Data  interface{} `json:"data"`
	Error *ErrorResp  `json:"error,omitempty"`
}

type ErrorResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RegisterReq struct {
	FullName string `json:"full_name" validate:"required"`
	Phone    string `json:"phone" validate:"required,e164"`
	Password string `json:"password" validate:"required,min=8"`
}

type VerifyOTPReq struct {
	Phone string `json:"phone" validate:"required"`
	Code  string `json:"code" validate:"required,len=6"`
}

type ResendOTPReq struct {
	Phone string `json:"phone" validate:"required"`
}

type LoginReq struct {
	Phone      string `json:"phone" validate:"required"`
	Password   string `json:"password" validate:"required"`
	RememberMe bool   `json:"remember_me"`
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ForgotReq struct {
	Phone string `json:"phone" validate:"required"`
}

type ResetPasswordReq struct {
	Phone       string `json:"phone" validate:"required"`
	Code        string `json:"code" validate:"required,len=6"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type TokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserResp struct {
    ID string `json:"id"`
    FullName string `json:"full_name"`
    Phone string `json:"phone"`
}
