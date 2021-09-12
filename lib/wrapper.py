# Function to sort an unsorted TaskCollect dictionary of tasks.
def tasksort(tasks):
    sorted_tasks = {}
    return sorted_tasks

# Function to shorten a dictionary to a certain number of values.
def shorten(tasks, amt):
    sorted_tasks = {}

    if len(tasks) > amt:
        for x in range(amt):
            sorted_tasks[x] = tasks[x]
        return sorted_tasks
    else:
        return tasks

# Function to convert user timetable data into an HTML component, ready for rendering.
def render_timetable(timetable, timetable2, lesson_list, lesson_list2, week, day):
    html_day = f"<div class=\"banner-strip\">{day}</div>"
    html_week = f"<h5 class=\"tiny-header\">{week}</h5>"
    count = 0
    html_timetable = ""
    html_timetable2 = ""
    for item in lesson_list:
        html_timetable = html_timetable + f"<div class = \"timetable-lesson\"><h5>{timetable[item][1]}</h5><h4>{item}</h4></div>"
    
    for item in lesson_list2:
        html_timetable2 = html_timetable2 + f"<div class = \"timetable-lesson\"><h5>{timetable2[item][1]}</h5><h4>{item}</h4></div>"
    
    html_tomorrow = f"<div class=\"banner-strip\">{timetable2[lesson_list2[-1]][0]}</div>"

    return html_day, html_week, html_timetable, html_tomorrow, html_timetable2

# Function to convert user message/email data into an HTML component, ready for rendering.
def render_msgs(msgs):
    html_msgs = '<br><p class="fg-red">Oopsy...</p>'
    return html_msgs

# Function to convert user task/assignment data into an HTML component, ready for rendering.
def render_tasks(tasks):
    html_tasks = '<br><p class="fg-red">Oopsy...</p>'
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
