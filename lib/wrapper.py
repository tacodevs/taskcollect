# Function to calculate a Python datetime object for a X days from today.
def date_from_now(days):
    return days

# Function to sort an unsorted TaskCollect dictionary of tasks.
def tasksort(tasks):
    return tasks

# Function to shorten a dictionary to a certain number of values.
def shorten(tasks, amt):
    return tasks
    
# Function to convert user timetable data into an HTML component, ready for rendering.
def render_timetable(timetable, timetable2, lesson_list, lesson_list2, week, day):
    html_day = f"<div class=\"banner-strip\">{day}</div>"
    html_week = f"<h5 class=\"tiny-header\">{week}</h5>"
    html_timetable = ""
    html_timetable2 = ""
    for item in lesson_list:
        html_timetable = html_timetable + f"<a href = 'https://daymap.gihs.sa.edu.au/DayMap/Student/plans/class.aspx?eid={timetable[item][2]}' ><div class = \"timetable-lesson\"><h5>{timetable[item][1]}</h5><h4>{item}</h4></div></a>"
    
    for item in lesson_list2:
        html_timetable2 = html_timetable2 + f"<a href = 'https://daymap.gihs.sa.edu.au/DayMap/Student/plans/class.aspx?eid={timetable2[item][2]}' ><div class = \"timetable-lesson\"><h5>{timetable2[item][1]}</h5><h4>{item}</h4></div></a>"
    
    html_tomorrow = f"<div class=\"banner-strip\">{timetable2[lesson_list2[-1]][0]}</div>"

    return html_day, html_week, html_timetable, html_tomorrow, html_timetable2

# Function to convert user message/email data into an HTML component, ready for rendering.
def render_msgs(msgs):
    html_msgs = ""
    for msg in msgs:
        html_msgs += f'<tr><td><a href="https://daymap.gihs.sa.edu.au/daymap/coms/Message.aspx?ID={msg}&via=4" target="_blank" rel="noopener noreferrer" class="msg-box no-focus-border"><div><div class="msgbox daymap-msgbox"><b>{msgs[msg][3]}</b><br><p>From {msgs[msg][2]} {msgs[msg][0]}</p><br><p>{msgs[msg][1]}</p></div></div></a></td></tr>' 
    return html_msgs

# Function to convert user task/assignment data into an HTML component, ready for rendering.
def render_tasks(tasks):
    html_tasks = ""
    for task in tasks:
        html_tasks += f'<tr><td {tasks[task][4]}><a href="https://9p.io/plan9/" target="_blank" rel="noopener noreferrer" class="boring-link no-focus-border"><div class="msgbox daymap-msgbox"><div><b>{tasks[task][0]} {tasks[task][3]}</b><br><p>{task} DUE {tasks[task][2]}</p></div></div></a></td></tr>'
    return html_tasks

# Function to convert user timetable data into a CSV.
def tocsv_timetable(tasks):
    csv = ''
    return csv

# Function to convert user message/email data into a CSV.
def tocsv_msgs(tasks):
    csv = ''
    return csv

# Function to convert user task/assignment data into a CSV.
def tocsv_tasks(tasks):
    csv = ''
    return csv
