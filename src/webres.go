package main

// TODO: Use external HTML files and templates for easier management

const htmlHead = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>`

const htmlNav = `</title>
<link rel="stylesheet" href="/css">
</head>
<body>
<nav>
<a href="/timetable">Timetable</a> |
<a href="/tasks">Tasks</a> |
<a href="/res">Resources</a> |
<a href="/logout">Logout</a>
</nav>
`

const htmlEnd = `</body>
</html>
`

const tasksHeader = `<h1>Tasks</h1>
<details>
<summary>Active tasks</summary>
<table>
<thead><tr>
<th class="due">Due date</th>
<th class="subject">Subject</th>
<th class="task">Task</th>
<th class="link">Source platform link</th>
</tr></thead>
`

const notDueHeader = `</table>
</details>
<details>
<summary>No due date</summary>
<table>
<thead><tr>
<th class="subject">Subject</th>
<th class="task">Task</th>
<th class="link">Source platform link</th>
</tr></thead>
`

const overdueHeader = `</table>
</details>
<details>
<summary>Overdue tasks</summary>
<table>
<thead><tr>
<th class="subject">Subject</th>
<th class="task">Task</th>
<th class="link">Source platform link</th>
</tr></thead>
`

const submittedHeader = `</table>
</details>
<details>
<summary>Submitted tasks</summary>
<table>
<thead><tr>
<th class="subject">Subject</th>
<th class="task">Task</th>
<th class="link">Source platform link</th>
</tr></thead>
`

const uploadHtml = `/upload">
<label for="file">Select file:</label>
<input type="file" name="file"><br>
<input type="submit" value="Upload file">
</form>
<h4>Remove file</h4>
<form action="/tasks/`

const loginHead = `Login</title>
<link rel="stylesheet" href="/css">
</head>
<body>
<form method="POST" enctype="application/x-www-form-urlencoded" action="/auth">
<h1>TaskCollect</h1>
<h3>Login</h3>
`

const loginBody = `<label for="school">School:</label><br>
<select id="school" name="school">
<option value="gihs">Glenunga International High School</option>
</select><br>
<label for="usr">Username:</label><br>
<input type="text" id="usr" name="usr"><br>
<label for="pwd">Password:</label><br>
<input type="password" id="pwd" name="pwd"><br>
<input type="submit" value="Login">
</form>
`

const loginPage = htmlHead + loginHead + loginBody + htmlEnd

const loginFailed = htmlHead + loginHead +
	"<h4>Authentication failed</h4>\n" + loginBody + htmlEnd

const notFoundPage = htmlHead +
	"404 Not Found" +
	htmlNav +
	`<h1>404 Not Found</h1>
<p>The requested resource was not found on the server.</p>
` + htmlEnd

const serverErrorPage = htmlHead +
	"500 Internal Server Error" +
	htmlNav +
	`<h1>500 Internal Server Error</h1>
<p>The server encountered an unexpected error and cannot continue.</p>
` + htmlEnd
