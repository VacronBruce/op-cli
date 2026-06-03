package testutil

import (
	"fmt"
	"net/http"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// MockClient implements api.APIClient for testing.
type MockClient struct {
	ProjectValue string

	RequireProjectFn      func() (string, error)
	GetFn                 func(path string, result interface{}) error
	PostFn                func(path string, body interface{}, result interface{}) error
	PatchFn               func(path string, body interface{}, result interface{}) error
	DoRawFn               func(method, href string) (*http.Response, error)
	GetWorkPackageFn      func(id int) (*api.WorkPackage, error)
	ListWorkPackagesFn    func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error)
	ListAllWorkPackagesFn func(filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error)
	SearchByJiraIDFn      func(jiraID string) (*api.WPCollection, error)
	CreateWorkPackageFn   func(project string, req *api.CreateWPRequest) (*api.WorkPackage, error)
	UpdateWorkPackageFn   func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error)
	ListVersionsFn        func(project string) (*api.VersionCollection, error)
	CreateVersionFn       func(req *api.CreateVersionRequest) (*api.Version, error)
	FindActiveSprintFn    func(project string) (*api.Version, error)
	ResolveVersionFn      func(project, name string) (*api.Version, error)
	ListProjectsFn        func() (*api.ProjectCollection, error)
	GetProjectFn          func(identifier string) (*api.Project, error)
	GetMeFn               func() (*api.User, error)
	UploadAttachmentFn    func(wpID int, filePath string, description string) (*api.Attachment, error)
	ListActivitiesFn      func(wpID int) (*api.ActivityCollection, error)
	PostCommentFn         func(wpID int, markdown string) error
	EditCommentFn         func(activityID int, markdown string) error
	CreateRelationFn      func(fromID int, relType string, toID int) error
}

func (m *MockClient) RequireProject() (string, error) {
	if m.RequireProjectFn != nil {
		return m.RequireProjectFn()
	}
	if m.ProjectValue != "" {
		return m.ProjectValue, nil
	}
	return "", fmt.Errorf("no project")
}

func (m *MockClient) Get(path string, result interface{}) error {
	if m.GetFn != nil {
		return m.GetFn(path, result)
	}
	return fmt.Errorf("Get not mocked")
}

func (m *MockClient) Post(path string, body interface{}, result interface{}) error {
	if m.PostFn != nil {
		return m.PostFn(path, body, result)
	}
	return fmt.Errorf("Post not mocked")
}

func (m *MockClient) Patch(path string, body interface{}, result interface{}) error {
	if m.PatchFn != nil {
		return m.PatchFn(path, body, result)
	}
	return fmt.Errorf("Patch not mocked")
}

func (m *MockClient) DoRaw(method, href string) (*http.Response, error) {
	if m.DoRawFn != nil {
		return m.DoRawFn(method, href)
	}
	return nil, fmt.Errorf("DoRaw not mocked")
}

func (m *MockClient) GetWorkPackage(id int) (*api.WorkPackage, error) {
	if m.GetWorkPackageFn != nil {
		return m.GetWorkPackageFn(id)
	}
	return nil, fmt.Errorf("GetWorkPackage not mocked")
}

func (m *MockClient) ListWorkPackages(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
	if m.ListWorkPackagesFn != nil {
		return m.ListWorkPackagesFn(project, filters, sortBy, pageSize)
	}
	return &api.WPCollection{}, nil
}

func (m *MockClient) ListAllWorkPackages(filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
	if m.ListAllWorkPackagesFn != nil {
		return m.ListAllWorkPackagesFn(filters, sortBy, pageSize)
	}
	return &api.WPCollection{}, nil
}

func (m *MockClient) SearchByJiraID(jiraID string) (*api.WPCollection, error) {
	if m.SearchByJiraIDFn != nil {
		return m.SearchByJiraIDFn(jiraID)
	}
	return &api.WPCollection{}, nil
}

func (m *MockClient) CreateWorkPackage(project string, req *api.CreateWPRequest) (*api.WorkPackage, error) {
	if m.CreateWorkPackageFn != nil {
		return m.CreateWorkPackageFn(project, req)
	}
	return nil, fmt.Errorf("CreateWorkPackage not mocked")
}

func (m *MockClient) UpdateWorkPackage(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
	if m.UpdateWorkPackageFn != nil {
		return m.UpdateWorkPackageFn(id, req)
	}
	return nil, fmt.Errorf("UpdateWorkPackage not mocked")
}

func (m *MockClient) ListVersions(project string) (*api.VersionCollection, error) {
	if m.ListVersionsFn != nil {
		return m.ListVersionsFn(project)
	}
	return &api.VersionCollection{}, nil
}

func (m *MockClient) CreateVersion(req *api.CreateVersionRequest) (*api.Version, error) {
	if m.CreateVersionFn != nil {
		return m.CreateVersionFn(req)
	}
	return nil, fmt.Errorf("CreateVersion not mocked")
}

func (m *MockClient) FindActiveSprint(project string) (*api.Version, error) {
	if m.FindActiveSprintFn != nil {
		return m.FindActiveSprintFn(project)
	}
	return nil, fmt.Errorf("no active sprint")
}

func (m *MockClient) ResolveVersion(project, name string) (*api.Version, error) {
	if m.ResolveVersionFn != nil {
		return m.ResolveVersionFn(project, name)
	}
	return nil, fmt.Errorf("ResolveVersion not mocked")
}

func (m *MockClient) ListProjects() (*api.ProjectCollection, error) {
	if m.ListProjectsFn != nil {
		return m.ListProjectsFn()
	}
	return &api.ProjectCollection{}, nil
}

func (m *MockClient) GetProject(identifier string) (*api.Project, error) {
	if m.GetProjectFn != nil {
		return m.GetProjectFn(identifier)
	}
	return nil, fmt.Errorf("GetProject not mocked")
}

func (m *MockClient) GetMe() (*api.User, error) {
	if m.GetMeFn != nil {
		return m.GetMeFn()
	}
	return nil, fmt.Errorf("GetMe not mocked")
}

func (m *MockClient) UploadAttachment(wpID int, filePath string, description string) (*api.Attachment, error) {
	if m.UploadAttachmentFn != nil {
		return m.UploadAttachmentFn(wpID, filePath, description)
	}
	return nil, fmt.Errorf("UploadAttachment not mocked")
}

func (m *MockClient) ListActivities(wpID int) (*api.ActivityCollection, error) {
	if m.ListActivitiesFn != nil {
		return m.ListActivitiesFn(wpID)
	}
	return &api.ActivityCollection{}, nil
}

func (m *MockClient) PostComment(wpID int, markdown string) error {
	if m.PostCommentFn != nil {
		return m.PostCommentFn(wpID, markdown)
	}
	return nil
}

func (m *MockClient) EditComment(activityID int, markdown string) error {
	if m.EditCommentFn != nil {
		return m.EditCommentFn(activityID, markdown)
	}
	return nil
}

func (m *MockClient) CreateRelation(fromID int, relType string, toID int) error {
	if m.CreateRelationFn != nil {
		return m.CreateRelationFn(fromID, relType, toID)
	}
	return nil
}
