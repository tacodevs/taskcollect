# Required to send HTTP requests
import requests

# Required to make NTLM handshakes
import requests_ntlm

# HTML parser; JS already has one
from lxml import html

# Required to print to standard error output
import sys

#required to parse HTML strings
import urllib.parse

#date/time management
import datetime

#json and csv file management
import json
import csv

# Function to authenticate and get a resource from DayMap via HTTP.
def daymap_get(webpage, username, password):
    URL_ROOT = 'https://daymap.gihs.sa.edu.au'

    # Sets up a cookie session.
    # THIS IS INSANELY IMPORTANT! AUTHENTICATION WON'T WORK WITHOUT IT!
    s = requests.Session()

    # Sends a GET to the desired page.
    s1 = s.get(webpage)

    # Gets the 'signin' parameter.
    signin_id = s1.url.split('?signin=')[-1]

    # Constructs a redirect URL using the 'signin' ID.
    URL_STAGE2 = f'{URL_ROOT}/daymapidentity/external?provider=Glenunga%20Windows%20Auth&signin={signin_id}'

    # Makes an NTLM transaction at the constructed URL.
    s2 = s.get(
        URL_STAGE2, 
        headers={
            'Referer': s1.url # Pretend this is like a redirect.
        },
        auth=requests_ntlm.HttpNtlmAuth(username, password) # Transaction credentials
    )

    if ('Daymap Identity Error' in s2.text):
        print('daymap.py: Got DayMap Identity Error!', file=sys.stderr)
        exit(1)

    # Raises an error if NTLM failed.
    if (s2.status_code != 200):
        print(f'daymap.py: Received error status code ({s2.status_code} {s2.reason}).', file=sys.stderr)
        print('daymap.py: Possibly invalid credentials.', file=sys.stderr)
        exit(1)

    if ('<form method="POST"' not in s2.text):
        print('daymap.py: Status code 200, but no HTML form found.', file=sys.stderr)
        exit(1)

    # Parses the HTML form using lxml.
    s3_tree = html.fromstring(s2.content)

    # Finds all inputs that aren't submits.
    inputs = s3_tree.xpath('//input[@type!="submit"]')

    post_payload = {}

    # Adds the inputs to the payload dictionary.
    for inp in inputs:
        post_payload[inp.name] = inp.value

    URL_STAGE3 = f'{URL_ROOT}/DaymapIdentity/was/client'

    # Pretends that the form is being submitted by manually POSTing there.
    s3 = s.post(URL_STAGE3, data=post_payload)

    # Checks if everything went correctly.
    if '<form' not in s3.text:
        print('daymap.py: No HTML form found.', file=sys.stderr)
        exit(1)

    # Parses the form using lxml.
    s4_tree = html.fromstring(s3.content)

    # Enumerates parameters from inputs.
    inputs = s4_tree.xpath('//input')

    post_payload = {}

    # Adds the inputs to the payload dictionary.
    for inp in inputs:
        post_payload[inp.name] = inp.value

    URL_STAGE4 = f'{URL_ROOT}/Daymap/'

    # Pretends that the form is being submitted by manually POSTing there.
    s4 = s.post(URL_STAGE4, data=post_payload)

    # At this point 's4' should contain the target document.
    # The HTTP response is stored in 's4.text'.
    return s4.text

# Function to get the timetable for a couple of days
def get_lessons(thing, username, password):

    #define lists and dictionaries: 2 sets, one for each day
    timetable = {}
    tomorrow_timetable = {}
    lesson_list = []
    lesson_list2 = []
    #lesson_info = open("./usr/daymap_html.txt", "w")
    #get html from daymap
    page_html = daymap_get("https://daymap.gihs.sa.edu.au/daymap/student/dayplan.aspx", username, password)

    #split lines
    page_html = page_html.split("\n")
    for line in page_html:
        if "ctl00_cp_divEvents" in line:
            break
    
    #take specific line to work with
    lesson_line = line

    #empties list to optimise
    page_html = []
    
    #looks for week class
    class_index = lesson_line.index("diaryWeek")

    #cuts it down to minimise searching
    lesson_line = lesson_line[class_index+10:]
    count = 0

    #finds the end of the related tag
    start_index = lesson_line.index(">")
    end_index = lesson_line.index("<")
    week = lesson_line[start_index+1:end_index]
    
    class_index = lesson_line.index("diaryDay")
    lesson_line = lesson_line[class_index+8:]

    start_index = lesson_line.index(">")
    end_index = lesson_line.index("<")

    today = lesson_line[start_index+1: end_index]
    
    #checks the day and tells it how many lessons there are on that day to look for
    if "Wednesday" in today:
        limit = 4
    elif ("Friday") in today:
        limit = 6
    else:
        limit = 5
    Daycount = 1

    #loop to find the lesson time and subject
    print(today)
    while Daycount <= limit:
        id_index = lesson_line.index('data-id=')
        lesson_line = lesson_line[id_index+9:]
        end_index = lesson_line.index('"')
        ID = str(lesson_line[:end_index])
        link = f"https://daymap.gihs.sa.edu.au/DayMap/Student/plans/class.aspx?eid={ID}"
        time_index = lesson_line.index("class='t'")

        lesson_line = lesson_line[time_index+9:]

        start_index = lesson_line.index(">")
        end_index = lesson_line.index("<")
        time = str(lesson_line[start_index + 1: end_index])
        #finds the subject class
        subject_index = lesson_line.index("event")

        lesson_line = lesson_line[subject_index+6:]

        start_index = lesson_line.index(">")
        end_index = lesson_line.index("<")

        subject = lesson_line[start_index + 1: end_index]
        lesson_list.append(subject[:-2])

        #adds it to a dictionary along with the date and time
        timetable[subject[:-2]] = [today, time, link]
        Daycount += 1
        #if statement to check if the next school day is monday
    print(timetable)
    try:
        class_index = lesson_line.index("diaryDay")
        lesson_line = lesson_line[class_index+10:]
        count = 0

        #basically the same as above
        for char in lesson_line:
            count += 1
            if char == ">":
                break
        lesson_line = lesson_line[count:]
        tomorrow = ""

        #basically the same as above
        for char in lesson_line:
            if char == "<":
                break
            tomorrow = tomorrow + char

        #same code from above repeated, only difference is saving to a different set of variables to minimise confusion
        if "Wednesday" in tomorrow:
            limit = 4
        elif ("Friday") in tomorrow:
            limit = 6
        else:
            limit = 5
        Daycount = 1
        while Daycount <= limit:
            id_index = lesson_line.index('data-id=')
            lesson_line = lesson_line[id_index+9:]
            end_index = lesson_line.index('"')
            ID = str(lesson_line[:end_index])

            link = f"https://daymap.gihs.sa.edu.au/DayMap/Student/plans/class.aspx?eid={ID}"
            
            time_index = lesson_line.index("class='t'")
            lesson_line = lesson_line[time_index+9:]

            start_index = lesson_line.index(">")
            end_index = lesson_line.index("<")
            time = str(lesson_line[start_index + 1: end_index])
            
            #finds the subject class
            subject_index = lesson_line.index("event")

            lesson_line = lesson_line[subject_index+6:]

            start_index = lesson_line.index(">")
            end_index = lesson_line.index("<")

            subject = lesson_line[start_index + 1: end_index]
            lesson_list2.append(subject[:-2])

            #adds it to a dictionary along with the date and time
            tomorrow_timetable[subject[:-2]] = [tomorrow, time, link]
            Daycount += 1
    except:
        None
        
    return week, timetable, tomorrow_timetable


# Function to get the specified user's DayMap messages.
def get_msgs(username, password):
    msgs = {}
    start_index = 0
    end_index = 0
    page_html = daymap_get("https://daymap.gihs.sa.edu.au/daymap/student/dayplan.aspx", username, password)
    page_html = page_html.split("\n")
    for line in page_html:
        if "<div class='Header'>Messages </div>" in line:
            break
        else:
            page_html.remove(line)
    index = page_html.index(line)
    page_html = page_html[index:]
    for line in page_html:
        if "Messages" in line:
            break
        else:
            page_html.remove(line)
    perm_line = line
    msg_count = 0
    try:
        while "message|" in perm_line and msg_count < 15:
            index = line.index("message|")
            line = line[index+8:]
            ID = ""
            end_index = line.index("'")
            ID = str(line[:end_index])
            perm_line = line
            msg_html = daymap_get(f"https://daymap.gihs.sa.edu.au/daymap/coms/Message.aspx?ID={ID}&via=4", username, password)
            with open("./usr/msg.txt", "w") as f:
                f.write(msg_html)
            msg_html = msg_html.split("\n")
            for line in msg_html:
                if "LabelRG msgSentOn" in line:
                    break
                else:
                    None
            index = line.index("LabelRG msgSentOn")
            line = line[index+17:]
            date = ""
            start_index = line.index(">")
            end_index = line.index("<")
            date = line[start_index+1:end_index]
            if "msgSubject" in line:
                index = line.index("msgSubject")
                line = line[index:]
                subject = ""
                start_index = line.index(">")
                end_index = line.index("<")
                subject = line[start_index+1:end_index]
            else:
                subject = str(ID)
            
            index = line.index("msgSender")
            line = line[index:]
            start_index = line.index(">")
            end_index = line.index("<")
            sender = line[start_index+1:end_index]
            index = line.index("msgBody")
            line = line[index:]
            start_index = line.index(">")
            end_index = line.index("<")
            body = line[start_index+1:end_index]
            msgs[ID] = [date, body, sender, subject]
            line = perm_line
            msg_count += 1
    except:
        None
    return msgs

# Function to get the specified user's tasks from DayMap.
def get_tasks(username, password):
    tasks = []

    #note that this code will cause the webpage to be slow, hence why there is a different section for this
    page_html = daymap_get("https://daymap.gihs.sa.edu.au/daymap/student/dayplan.aspx", username, password)
    file = open("./usr/tasks_html", "w")
    file.write(page_html)
    page_html = page_html.split("\n")
    page_html.remove("")
    for line in page_html:
        if "ctl00_cp_divAssignments" not in line:
            page_html.remove(line)
        else:
            break 
    index = page_html.index(line)
    page_html = page_html[index:]
    for line in page_html:
        while ("#68739B" or "#FF4E1F") in line:
            if ("#68739B" or "#FF4E1F") in line:
                if ("#FF4E1F" in line) and ("#68739B" in line):
                    index1 = line.index("#FF4E1F")
                    index2 = line.index ("#68739B")
                    if index1 < index2:
                        overdue = "class = 'err-bg'"
                        index = line.index("#FF4E1F")
                    else:
                        overdue = ""
                        index = line.index("#68739B")
                elif "#68739B" in line:
                    overdue = ""
                    index = line.index("#68739B")
                elif "#FF4E1F" in line:
                    overdue = "class = 'err-bg'"
                    index = line.index("#FF4E1F")
                line = line[index:]
                index = line.index("OpenTask")
                line = line[index+8:]
                start_index = line.index("(")
                end_index = line.index(")")
                ID = str(line[start_index+1:end_index])
                
                index = line.index("class='cap'")
                line = line[index:]
                start_index = line.index(">")
                end_index = line.index("<")
                subject = line[start_index+1:end_index]
                line = line[end_index+1:]
                start_index = line.index(">")
                line = line[start_index+1:]
                start_index = line.index(">")
                line = line[start_index:]
                start_index = line.index(">")
                end_index = line.index("<")
                sender = line[start_index+1:end_index] 
                line = line[len(sender)+6:]
                end_index = line.index("<")
                due = str(line[:end_index])
                
                index = line.index("Caption")
                line = line[index:]
                
                start_index = line.index(">")
                end_index = line.index("<")
                assessment_type = line[start_index+1:end_index]
                line = line[len(assessment_type)+20:]
                end_index = line.index("<")
                task_name = line[:end_index]
                tasks.append([task_name, subject, "No description.", f"https://daymap.gihs.sa.edu.au/daymap/student/assignment.aspx?TaskID={ID}&d=1", due, None, overdue, "daymap"])
    return tasks

    

    
