# Required for Python datetime object usage.
import datetime

# Function to calculate a Python datetime object for a X days from today.
def date_from_now(days):
    altdatetime = datetime.datetime.utcnow()
    altdatetime += datetime.timedelta(days=1)
    return altdatetime

# Function to sort an unsorted TaskCollect dictionary of messages.
def msgsort(msgs):
    return msgs

# Function to sort an unsorted TaskCollect dictionary of tasks.
def tasksort(tasks):
    return tasks

# Function to shorten a dictionary to a certain number of values.
def dictshorten(tasks, amt):
    sorted_tasks = {}

    if len(tasks) > amt:

        i = 1

        for t in tasks:

            sorted_tasks[t] = tasks[t]

            if i < amt:
                i += 1
            else:
                break

        return sorted_tasks

    else:
        return tasks

# Function to shorten a list to a certain number of values.
def listshorten(tasks, amt):
    sorted_tasks = []

    if len(tasks) > amt:
        for x in range(amt):
            sorted_tasks.append(tasks[x])
        return sorted_tasks
    else:
        return tasks
    
# Function to convert user timetable data into an HTML component, ready for rendering.
def render_timetable(timetable):

    html_timetable = '<p class="fg-red">You did a big oopsy and you didn\'t even know it.</p>'

    return html_timetable

# Function to convert user message data into an HTML component, ready for rendering.
def render_msgs(msgs):

    html_msgs = ""

    for msg in msgs:
        html_msgs += f'<tr><td><a href="https://daymap.gihs.sa.edu.au/daymap/coms/Message.aspx?ID={msg}&via=4" '
        html_msgs += 'target="_blank" rel="noopener noreferrer" class="boring-link no-focus-border">'
        html_msgs += f'<div class="msgbox daymap-msgbox"><div><b>{msgs[msg][3]}</b><br>'
        html_msgs += f'<p><b>From {msgs[msg][2]} ({msgs[msg][0]})</b><br>{msgs[msg][1]}</p></div></div></a></td></tr>'

    return html_msgs

# Function to convert user task/assignment data into an HTML component, ready for rendering.
def render_tasks(tasks):

    html_tasks = ""

    for task in tasks:

        # TODO: We need to consider if certain data fields are empty and render accordingly.

        if task[6]:
            isoverdue = ' class="err-bg"'
        else:
            isoverdue = ''

        html_tasks += f'<tr><td{isoverdue}><a href="{task[3]}" target="_blank" rel="noopener noreferrer" '
        html_tasks += f'class="boring-link no-focus-border"><div class="msgbox {task[7]}-msgbox">'
        html_tasks += f'<div><b>{task[0]}</b><br><p><b>{task[1]}</b><br>{task[2]}</p>'
        html_tasks += '</div></div></a></td></tr>'

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