# Required for session token lifetime generation
import datetime
from re import U

# Required to decode the user input from HTML query
import urllib.parse

# Required to set up a simple HTTP server
from http.server import HTTPServer, BaseHTTPRequestHandler

# Imports the DayMap TaskCollect driver
from lib import daymap

# Imports the Google Classroom TaskCollect driver
from lib import classroom

# Imports the Edpuzzle TaskCollect driver
from lib import edpuzzle

# Imports the Stile TaskCollect driver
from lib import stile

# Imports the Exchange Web Services driver
from lib import ews

# Imports the TaskCollect wrapper module
from lib import wrapper

# Required to check if HTTP-requested files exist
import os

# Required to create randomised session tokens
import random

# Required to print to standard error output
import sys

# Required to output detailed Python error messages
import traceback

# The server request handler.
class Handler(BaseHTTPRequestHandler):
    def do_GET(self):

        # Replaces the default server information with less revealing information.
        self.server_version = "TaskCollect"
        self.sys_version = "(GNU/Linux)"

        try:
            # Splits the requested path into the requested resource and queries.
            if '?' in self.path:
                http_res = self.path[:self.path.index("?")]
                http_query = self.path[self.path.index("?")+1:]
            else:
                http_res = self.path
                http_query = False
            
            # If there is a query, process the query into a Python dictionary.
            if http_query != False:
                query = dict(kvpair.split("=") for kvpair in http_query.split("&"))

            special_reslist = [
                "/",
                "/index",
                "/login",
                "/login-err",
                "daymap-tasks.csv",
                "classroom-tasks.csv",
                "edpuzzle-tasks.csv",
                "stile-tasks.csv",
                "tasks.csv"
            ]

            # Login handling. Has to be before session token checking.
            if http_res == "/login":

                # If there are no queries, simply send an HTTP response with "/login".
                if not http_query:
                    self.httpsrv("/login.html", "text/html")

                # Attempt to log the user on and provide a session token.
                else:

                    if "username" in query and "password" in query:

                        with open("./usr/creds.csv", "r") as credsfile:

                            query["username"] = urllib.parse.unquote(query["username"])
                            query["password"] = urllib.parse.unquote(query["password"])

                            usr_correct = False
                            pwd_correct = False
                            credlnnum = 0
                            othercreds = ""

                            # TODO: Register and sign on CURRIC-registered users who are not in the credentials database.
                            # Checks if the username and password are correct.
                            for line in credsfile:

                                cred = line.split(",")

                                if len(cred) == 5:

                                    if cred[2] == query["username"]:

                                        usr_correct = True

                                        if cred[3] == query["password"]:
                                            pwd_correct = True
                                            authcred = cred

                                    else:
                                        othercreds += line

                            # If the username and password are correct, redirect the
                            # user to the home page and provide a session token.
                            if usr_correct and pwd_correct:

                                self.send_response(302)
                                self.send_header("Location", "/")

                                randtoken = ""
                                i = 0

                                while i != 40:
                                    randint = random.randint(48, 122)
                                    if randint not in range(57, 65) and randint not in range(90, 97):
                                        randtoken += chr(randint)
                                        i += 1
                                
                                # Sets a session expiry date of one day as of now.
                                expiretime = datetime.datetime.utcnow()
                                expiretime += datetime.timedelta(days=1)
                                tokenexpires = expiretime.strftime("%a, %d %b %Y %H:%M:%S GMT")

                                # Replaces the old session token with the new one in './usr/creds.csv'.
                                with open("./usr/creds.csv", "w") as f:
                                    authcred[4] = randtoken
                                    othercreds += ("\n" + ",".join(authcred))
                                    f.write(othercreds)

                                # TODO: In a production environment all cookies must have "Secure;" to enforce HTTPS!
                                self.send_header("Set-Cookie", f"session_id={randtoken}; expires={tokenexpires}; path=/; SameSite=Strict")
                                self.end_headers()

                            # BUG: The user does not, currently, get an error message if authentication fails.
                            # If the username and/or password are incorrect, reprovide "login.html".
                            else:
                                self.send_response(302)
                                self.send_header("Location", "/login?err=401")
                                self.end_headers()

                    # If there has been a previous login error, display a login error.
                    elif "err" in query:

                        self.send_response(401)
                        self.send_header("Content-type", "text/html")
                        self.end_headers()

                        with open("./web/login.html") as f:

                            file = ""

                            for l in f:
                                l1 = l.replace('<taskcollect plchold="login-status" />',
                                               '<p class="fg-red"><b>Incorrect username and/or password</b></p><br>')
                                file += l1

                            self.wfile.write(bytes(file, "utf-8"))

                    # If the username and/or password are incorrect, reprovide "login.html".
                    else:
                        self.send_response(302)
                        self.send_header("Location", "/login?err=401")
                        self.end_headers()

            # If the requested resource exists, provide it to the user.
            # These are resources which cannot be hindered by session token checking,
            # so are also placed above the token-checking section.

            elif os.path.isfile(f"./web{http_res}") and http_res not in special_reslist:
                if http_res[-5:] == ".html":
                    self.httperr(404)
                elif http_res[-4:] == ".css":
                    self.httpsrv(f"/{http_res}", "text/css")
                elif http_res[-4:] == ".png":
                    self.httpsrv(f"/{http_res}", "image/png")
                elif http_res[-4:] == ".svg":
                    self.httpsrv(f"/{http_res}", "image/svg+xml")

            else:

                # If the (non-login) request does not have any cookies, redirect to "/login".
                if "Cookie" not in self.headers:
                    self.send_response(302)
                    self.send_header("Location", "/login")
                    self.end_headers()

                # If the request has cookies, parse cookies and check if there is a valid session token.
                else:

                    # Cookie parsing.
                    for header in self.headers:
                        if header == "Cookie":
                            try:
                                cookies = self.headers["Cookie"]
                                cookies = dict(cookie.split("=") for cookie in cookies.split("; "))
                            except:
                                del cookies
                                self.httperr(400)
                                break

                    # Checks if the session token is valid.
                    if "session_id" in cookies:

                        with open("./usr/creds.csv", "r") as credsfile:

                            creds_correct = False

                            for line in credsfile:
                                cred = line.split(",")
                                if len(cred) == 5:
                                    if cookies["session_id"] == cred[4]:
                                        creds_correct = True
                                        username = cred[2]
                                        password = cred[3]
                                        break

                        # If the session token is valid, provide the requested resource.
                        if creds_correct:

                            # If the requested HTML resource exists, provide it to the user.
                            if os.path.isfile(f"./web{http_res}.html") and http_res not in special_reslist:
                                self.httpsrv(f"/{http_res}.html", "text/html")

                            # If the requested resource is the timetable CSV, generate a personalised version and send it.
                            elif http_res == "timetable.csv":

                                self.send_response(200)
                                self.send_header("Content-type", "text/csv")
                                self.end_headers()

                                daymap.get_lessons(username, password)
                                csv = wrapper.tocsv_timetable(timetable)

                                self.wfile.write(bytes(csv, "utf-8"))

                            # If the requested resource is the DayMap messages CSV, generate a personalised version and send it.
                            elif http_res == "daymap-msgs.csv":

                                self.send_response(200)
                                self.send_header("Content-type", "text/csv")
                                self.end_headers()

                                msgs = daymap.get_msgs(username, password)
                                csv = wrapper.tocsv_msgs(msgs)

                                self.wfile.write(bytes(csv, "utf-8"))

                            # If the requested resource is the EWS emails CSV, generate a personalised version and send it.
                            elif http_res == "ews-msgs.csv":

                                self.send_response(200)
                                self.send_header("Content-type", "text/csv")
                                self.end_headers()

                                msgs = ews.get_emails(username, password)
                                csv = wrapper.tocsv_msgs(msgs)

                                self.wfile.write(bytes(csv, "utf-8"))

                            # If the requested resource is the messages CSV, generate a personalised version and send it.
                            elif http_res == "msgs.csv":

                                self.send_response(200)
                                self.send_header("Content-type", "text/csv")
                                self.end_headers()

                                msgs = {}
                                
                                msgs.update(
                                    daymap.get_msgs(
                                        username, password
                                    )
                                )

                                msgs.update(
                                    ews.get_emails(
                                        username, password
                                    )
                                )

                                csv = wrapper.tocsv_msgs(msgs)

                                self.wfile.write(bytes(csv, "utf-8"))

                            # If the requested resource is the DayMap tasks CSV, generate a personalised version and send it.
                            elif http_res == "daymap-tasks.csv":

                                self.send_response(200)
                                self.send_header("Content-type", "text/csv")
                                self.end_headers()

                                tasks = daymap.get_tasks(username, password)
                                tasks = wrapper.tasksort(tasks)
                                csv = wrapper.tocsv_tasks(tasks)

                                self.wfile.write(bytes(csv, "utf-8"))

                            # If the requested resource is the Google Classroom tasks CSV, generate a personalised version and send it.
                            elif http_res == "classroom-tasks.csv":

                                self.send_response(200)
                                self.send_header("Content-type", "text/csv")
                                self.end_headers()

                                tasks = classroom.get_tasks(username, password)
                                tasks = wrapper.tasksort(tasks)
                                csv = wrapper.tocsv_tasks(tasks)

                                self.wfile.write(bytes(csv, "utf-8"))

                            # If the requested resource is the Edpuzzle tasks CSV, generate a personalised version and send it.
                            elif http_res == "edpuzzle-tasks.csv":

                                self.send_response(200)
                                self.send_header("Content-type", "text/csv")
                                self.end_headers()

                                tasks = edpuzzle.get_tasks(username, password)
                                tasks = wrapper.tasksort(tasks)
                                csv = wrapper.tocsv_tasks(tasks)

                                self.wfile.write(bytes(csv, "utf-8"))

                            # If the requested resource is the Stile tasks CSV, generate a personalised version and send it.
                            elif http_res == "stile-tasks.csv":

                                self.send_response(200)
                                self.send_header("Content-type", "text/csv")
                                self.end_headers()

                                tasks = stile.get_tasks(username, password)
                                tasks = wrapper.tasksort(tasks)
                                csv = wrapper.tocsv_tasks(tasks)

                                self.wfile.write(bytes(csv, "utf-8"))

                            # If the requested resource is the tasks CSV, generate a personalised version and send it.
                            elif http_res == "tasks.csv":

                                self.send_response(200)
                                self.send_header("Content-type", "text/csv")
                                self.end_headers()

                                tasks = {}

                                tasks.update(
                                    daymap.get_tasks(
                                        username, password
                                    )
                                )

                                tasks.update(
                                    classroom.get_tasks(
                                        username, password
                                    )
                                )

                                tasks.update(
                                    edpuzzle.get_tasks(
                                        username, password
                                    )
                                )

                                tasks.update(
                                    stile.get_tasks(
                                        username, password
                                    )
                                )

                                tasks = wrapper.tasksort(tasks)
                                csv = wrapper.tocsv_tasks(tasks)

                                self.wfile.write(bytes(csv, "utf-8"))

                            # If the requested resource is "/", provide and personalise it.
                            elif http_res == "/":

                                self.send_response(200)
                                self.send_header("Content-type", "text/html")
                                self.end_headers()

                                # Gets the user's timetable for the next two days, from DayMap.

                                html_week, html_today, timetable, lessons, timetable2, lessons2 = daymap.get_lessons(username, password)

                                # Collects the user's messages and emails into one dictionary, then cuts
                                # off excess messages that won't fit into the HTML document.

                                msgs = daymap.get_msgs(username, password)

                                msgs.update(
                                    ews.get_emails(
                                        username, password
                                    )
                                )
                                
                                msgs = wrapper.tasksort(msgs)
                                msgs = wrapper.shorten(msgs, 5)
                                
                                # Collects and combines all the user's tasks from each platform into one
                                # dictionary, then sorts and cuts off excess tasks that won't fit into
                                # the HTML document.

                                tasks = daymap.get_tasks(username, password)

                                tasks.update(
                                    classroom.get_tasks(
                                        username, password
                                    )
                                )

                                tasks.update(
                                    edpuzzle.get_tasks(
                                        username, password
                                    )
                                )

                                tasks.update(
                                    stile.get_tasks(
                                        username, password
                                    )
                                )

                                tasks = wrapper.tasksort(tasks)
                                tasks = wrapper.shorten(tasks, 5)
                                
                                #get JSON data from daymap
                                #daymap.get_daymapID(username, password)
                                #this is commented out because not needed yet
                        
                                # Convert user data to HTML components for rendering.
                                html_today, html_week, html_timetable, html_tomorrow, html_timetable2 = wrapper.render_timetable(timetable, timetable2, lessons, lessons2,  html_week, html_today)
                                html_msgs = wrapper.render_msgs(msgs)
                                html_tasks = wrapper.render_tasks(tasks)


                                tasks = wrapper.tasksort(tasks)
                                tasks = wrapper.shorten(tasks, 5)
                                
                                #get JSON data from daymap
                                #daymap.get_daymapID(username, password)
                                #this is commented out because not needed yet
                        
                                # Convert user data to HTML components for rendering.
                                html_today, html_week, html_timetable, html_tomorrow, html_timetable2 = wrapper.render_timetable(timetable, timetable2, lessons, lessons2,  html_week, html_today)
                                html_msgs = wrapper.render_msgs(daymap_msgs)
                                html_tasks = wrapper.render_tasks(daymap_tasks)

                                # Return the HTML document to the user, replacing placeholders with personalised HTML components.
                                with open("./web/index.html") as f:

                                    file = ""

                                    for l in f:
                                        l1 = l.replace('<taskcollect plchold="timetable" />', html_timetable)
                                        l2 = l1.replace('<taskcollect plchold="timetable2" />', html_timetable2)
                                        l3 = l2.replace('<taskcollect plchold="tomorrow" />', html_tomorrow)
                                        l4 = l3.replace('<taskcollect plchold="week" />', html_week)
                                        l5 = l4.replace('<taskcollect plchold="today" />', html_today)
                                        l6 = l5.replace('<taskcollect plchold="messages" />', html_msgs)
                                        l7 = l6.replace('<taskcollect plchold="tasks" />', html_tasks)
                                        file += l7

                                    self.wfile.write(bytes(file, "utf-8"))

                            # If the resource doesn't exist, raise error 404.
                            else:
                                self.httperr(404)

                        # If there is no valid session token, redirect to "/login".
                        else:
                            self.send_response(302)
                            self.send_header("Location", "/login")
                            self.end_headers()

                    # If there is no valid session token, redirect to "/login".
                    else:
                        self.send_response(302)
                        self.send_header("Location", "/login")
                        self.end_headers()

        except Exception:
            print("srv.py: 500 Internal Server Error!", file=sys.stderr)
            print(traceback.format_exc(), file=sys.stderr)
            self.httperr(500)

    # A function to provide an HTML file with a certain status code and content-type header.
    def httpsrv(self, srvfpath, mimetype, statcode=200):
        self.send_response(statcode)
        self.send_header("Content-type", mimetype)
        self.end_headers()

        with open(f"./web{srvfpath}", "rb") as f:
            self.wfile.write(f.read())

    # A function to send an HTTP response of with a specified status code,
    # with a corresponding HTML file.
    def httperr(self, errno):
        str_errno = str(errno)
        self.httpsrv(f"/err/{str_errno}.html", "text/html", statcode=errno)

print("srv.py: You can now access TaskCollect at 'http://localhost:1111'")

# Starts a server on port 1111 using the server handler declared earlier.
server = HTTPServer(("localhost", 1111), Handler)
server.serve_forever()
