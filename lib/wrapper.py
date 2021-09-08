# Function to calculate a Python datetime object for a X days from today.
def date_from_now(days):
    return days

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
def render_timetable(timetable):
    html_timetable = '<br><p class="fg-red">Oopsy...</p>'
    return html_timetable

# Function to convert user message/email data into an HTML component, ready for rendering.
def render_msgs(msgs):
    html_msgs = '<br><p class="fg-red">Oopsy...</p>'
    return html_msgs

# Function to convert user task/assignment data into an HTML component, ready for rendering.
def render_tasks(tasks):
    html_tasks = '<br><p class="fg-red">Oopsy...</p>'
    return html_tasks

# Function to convert user task/assignment data into a CSV.
def tocsv_tasks(tasks):
    csv = ''
    return csv