package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type HttpRequest struct {
	token  string
	userID int
}

func NewHttpRequest() *HttpRequest {
	return &HttpRequest{}
}
func (r *HttpRequest) LoginScele(username, password string) (token string, err error) {
	url := fmt.Sprintf("https://scele.cs.ui.ac.id/login/token.php?service=moodle_mobile_app&username=%s&password=%s", username, password)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	tokenStruct := Token{}
	err = json.NewDecoder(resp.Body).Decode(&tokenStruct)

	if tokenStruct.Token == "" {
		err = fmt.Errorf("wrong credentials")
	}
	token = tokenStruct.Token
	return
}

func (r *HttpRequest) RequestScele(wsfunction string, args map[string]interface{}, model interface{}) (err error) {
	url := fmt.Sprintf("https://scele.cs.ui.ac.id/webservice/rest/server.php?moodlewsrestformat=json&wstoken=%s&wsfunction=%s", r.token, wsfunction)
	for k, v := range args {
		url = fmt.Sprintf("%s&%s=%v", url, k, v)
	}
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&model)
	return
}

func (r *HttpRequest) GetSceleId() (userID int, err error) {
	sceleUser := SceleUser{}
	err = r.RequestScele("core_webservice_get_site_info", nil, &sceleUser)
	userID = sceleUser.SceleID
	return
}

func (r *HttpRequest) GetCourses() (courses []Course, err error) {
	courses = make([]Course, 0)
	err = r.RequestScele("core_enrol_get_users_courses", map[string]interface{}{"userid": r.userID}, &courses)
	return
}

func (r *HttpRequest) GetCourseDetail(courseID int) (resource []CourseResource, err error) {
	resource = make([]CourseResource, 0)
	err = r.RequestScele("core_course_get_contents", map[string]interface{}{"courseid": courseID}, &resource)

	/* sanitizedResources = make([]CourseResource, 0, len(resource))
	for _, r := range resource {
		if r.Uservisible && r.Visible == 1 {
			sanitizedModulesResource := make([]ModulesResource, 0, len(r.Modules))

			for _, m := range r.Modules {
				if m.Uservisible && m.Visible == 1 && m.Visibleoncoursepage == 1 {
					sanitizedModulesResource = append(sanitizedModulesResource, m)
				}
			}
			r.Modules = sanitizedModulesResource
			sanitizedResources = append(sanitizedResources, r)
		}
	} */
	return
}

func (r *HttpRequest) GetForumDiscusstion(forumID, page int) (forum Forum, err error) {
	forum = Forum{}
	err = r.RequestScele("mod_forum_get_forum_discussions_paginated", map[string]interface{}{"forumid": forumID, "page": page, "perpage": 10}, &forum)
	return
}
