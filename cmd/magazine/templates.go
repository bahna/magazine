package main

import (
	"html/template"
	"log"
	"path"
)

// passwordResetTmpl is served at /restore endpoint.
var passwordResetTmpl = template.Must(template.New("").Parse(`Your username: {{ .FirstName }} {{ .LastName }}
New password: {{ .Password }}

Please, use the newly generated password to access your user's account.
`))

// mailchimpNotificationTmpl is a template for sending a message to
// administrators.
var mailchimpNotificationTmpl = template.Must(template.New("").Parse(`Новый подписчик на новости

ИД рассылки: {{ index . "list_id" }}
Почта: {{ index . "email_address" }}
Статус: {{ index . "status" }}
Язык: {{ index . "language" }}
`))

func generateTmpls(tmplDir string, funcMap template.FuncMap) map[string]*template.Template {
	adminMasterTmpl := template.Must(template.ParseFiles(
		path.Join(tmplDir, "admin_base.html"),
	))

	masterTmpl := template.Must(template.ParseFiles(
		path.Join(tmplDir, "base.html"),
	))

	adminTmpls := map[string][]string{
		"admin/index": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_index.html"),
		},
		"admin/topics/index": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_topics.html"),
		},
		"admin/topics/new": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_new_topic.html"),
		},
		"admin/topics/edit": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_edit_topic.html"),
		},
		"admin/content/index": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_content.html"),
		},
		"admin/content/new": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_new_content.html"),
		},
		"admin/content/edit": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_edit_content.html"),
		},
		"admin/users/index": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_users.html"),
		},
		"admin/users/new": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_new_user.html"),
		},
		"admin/users/edit": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_edit_user.html"),
		},
		"admin/users/passchange": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_user_pass_change.html"),
		},
		"admin/files/list": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_files.html"),
		},
		"admin/files/edit": []string{
			path.Join(tmplDir, "admin_header.html"),
			path.Join(tmplDir, "admin_sidebar.html"),
			path.Join(tmplDir, "admin_edit_file.html"),
		},
	}

	tmpls := map[string][]string{
		"signup": []string{
			path.Join(tmplDir, "header.html"),
			path.Join(tmplDir, "footer.html"),
			path.Join(tmplDir, "signup.html"),
		},
		"index": []string{
			path.Join(tmplDir, "header.html"),
			path.Join(tmplDir, "footer.html"),
			path.Join(tmplDir, "index.html"),
		},
		"topic_index": []string{
			path.Join(tmplDir, "header.html"),
			path.Join(tmplDir, "footer.html"),
			path.Join(tmplDir, "topic_index.html"),
		},
		"login": []string{
			path.Join(tmplDir, "header.html"),
			path.Join(tmplDir, "footer.html"),
			path.Join(tmplDir, "login.html"),
		},
		"restore_access": []string{
			path.Join(tmplDir, "header.html"),
			path.Join(tmplDir, "footer.html"),
			path.Join(tmplDir, "restore_access.html"),
		},
		"material": []string{
			path.Join(tmplDir, "header.html"),
			path.Join(tmplDir, "footer.html"),
			path.Join(tmplDir, "material.html"),
		},
		"topic": []string{
			path.Join(tmplDir, "header.html"),
			path.Join(tmplDir, "footer.html"),
			path.Join(tmplDir, "topic.html"),
		},
		"subscription_done": []string{
			path.Join(tmplDir, "header.html"),
			path.Join(tmplDir, "footer.html"),
			path.Join(tmplDir, "subscription_done.html"),
		},
	}

	var t *template.Template
	var m = make(map[string]*template.Template)
	var err error

	for k, v := range adminTmpls {
		if t, err = template.Must(adminMasterTmpl.Clone()).Funcs(funcMap).ParseFiles(v...); err != nil {
			log.Fatal(err)
		}
		m[k] = t
	}

	for k, v := range tmpls {
		if t, err = template.Must(masterTmpl.Clone()).Funcs(funcMap).ParseFiles(v...); err != nil {
			log.Fatal(err)
		}
		m[k] = t
	}

	return m
}
