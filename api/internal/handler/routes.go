package handler

import (
	"net/http"

	"ai-gozero-agent/api/internal/middleware"
	"ai-gozero-agent/api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	jwtMw := middleware.NewJwtMiddleware(serverCtx.Config.Auth.AccessSecret)
	adminMw := middleware.NewAdminMiddleware()

	publicRoutes := []rest.Route{
		{
			Method:  http.MethodPost,
			Path:    "/api/auth/login",
			Handler: AuthLoginHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/auth/register",
			Handler: AuthRegisterHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/uploads/:category/:filename",
			Handler: UploadFileHandler(serverCtx),
		},
	}

	studentRoutes := []rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/auth/profile",
			Handler: AuthProfileHandler(serverCtx),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/auth/password",
			Handler: AuthPasswordHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/student/profile",
			Handler: StudentProfileGetHandler(serverCtx),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/student/profile",
			Handler: StudentProfileUpdateHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/student/resume/upload",
			Handler: StudentResumeUploadHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/student/profile-change-requests",
			Handler: StudentProfileChangeRequestsHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/student/profile-change-requests",
			Handler: StudentProfileChangeRequestCreateHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/student/employments",
			Handler: EmploymentListHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/student/employments",
			Handler: EmploymentCreateHandler(serverCtx),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/student/employments/:id",
			Handler: EmploymentUpdateHandler(serverCtx),
		},
		{
			Method:  http.MethodDelete,
			Path:    "/api/student/employments/:id",
			Handler: EmploymentDeleteHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/student/employments/:id/evidences",
			Handler: EmploymentEvidenceUploadHandler(serverCtx),
		},
		{
			Method:  http.MethodDelete,
			Path:    "/api/student/employments/:id/evidences/:evidenceId",
			Handler: EmploymentEvidenceDeleteHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/interview/sessions",
			Handler: InterviewSessionCreateHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/interview/sessions",
			Handler: InterviewSessionListHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/interview/sessions/:id",
			Handler: InterviewSessionDetailHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/interview/chat/sse",
			Handler: InterviewChatSSEHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/interview/chat/sse",
			Handler: InterviewChatSSEHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/interview/sessions/:id/messages",
			Handler: InterviewSessionMessagesHandler(serverCtx),
		},
		{
			Method:  http.MethodDelete,
			Path:    "/api/interview/sessions/:id",
			Handler: InterviewSessionDeleteHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/interview/sessions/:id/end",
			Handler: InterviewSessionEndHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/analysis/sessions/:id",
			Handler: AnalysisSessionHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/analysis/overview",
			Handler: AnalysisOverviewHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/analysis/trend",
			Handler: AnalysisTrendHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/analysis/generate/:sessionId",
			Handler: AnalysisGenerateHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/suggestions",
			Handler: SuggestionListHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/suggestions/generate",
			Handler: SuggestionGenerateHandler(serverCtx),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/suggestions/:id/read",
			Handler: SuggestionReadHandler(serverCtx),
		},
	}

	adminRoutes := []rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/admin/dashboard/stats",
			Handler: AdminDashboardStatsHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/admin/dashboard/employment-rate",
			Handler: AdminDashboardEmploymentRateHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/admin/dashboard/major-stats",
			Handler: AdminDashboardMajorStatsHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/admin/dashboard/trend",
			Handler: AdminDashboardTrendHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/admin/students",
			Handler: AdminStudentsListHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/admin/students/:id",
			Handler: AdminStudentsDetailHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/admin/students",
			Handler: AdminStudentsCreateHandler(serverCtx),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/admin/students/:id",
			Handler: AdminStudentsUpdateHandler(serverCtx),
		},
		{
			Method:  http.MethodDelete,
			Path:    "/api/admin/students/:id",
			Handler: AdminStudentsDeleteHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/admin/students/import",
			Handler: AdminStudentsImportHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/admin/profile-change-requests",
			Handler: AdminProfileChangeRequestsListHandler(serverCtx),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/admin/profile-change-requests/:id/review",
			Handler: AdminProfileChangeRequestReviewHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/admin/knowledge/upload",
			Handler: KnowledgeUploadHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/admin/knowledge",
			Handler: AdminKnowledgeListHandler(serverCtx),
		},
		{
			Method:  http.MethodDelete,
			Path:    "/api/admin/knowledge/:id",
			Handler: AdminKnowledgeDeleteHandler(serverCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/admin/employments",
			Handler: AdminEmploymentsListHandler(serverCtx),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/admin/employments/:id/status",
			Handler: AdminEmploymentStatusUpdateHandler(serverCtx),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/admin/employments/:id/review",
			Handler: AdminEmploymentReviewHandler(serverCtx),
		},
	}

	server.AddRoutes(publicRoutes)
	server.AddRoutes(rest.WithMiddleware(jwtMw.Handle, studentRoutes...))
	server.AddRoutes(rest.WithMiddlewares([]rest.Middleware{jwtMw.Handle, adminMw.Handle}, adminRoutes...))
}
