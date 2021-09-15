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
def get_lessons(username, password):

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

    #take specific line to work with
    lesson_line = page_html[211]

    #empties list to optimise
    page_html = []
    
    #looks for week class
    class_index = lesson_line.find("diaryWeek")

    #cuts it down to minimise searching
    lesson_line = lesson_line[class_index+10:-1]
    count = 0

    #finds the end of the related tag
    for char in lesson_line:
        count += 1
        if char == ">":
            break

    #cuts down again to minimise searching for next step
    lesson_line = lesson_line[count:-1]

    #sets variable for week
    week = ""
    for char in lesson_line:

        #loops until the start of the next tag i.e. when the text stops
        if char == "<":
            break
        week = week + char
    
    #today var is set
    today = ""

    #looks for the day class
    class_index = lesson_line.find("diaryDay")

    #cuts down to minimise searching
    lesson_line = lesson_line[class_index+9:-1]
    count = 0

    #finds the end of the related tag
    for char in lesson_line:
        count += 1
        if char == ">":
            break

    #cuts down again to minimise searching
    lesson_line = lesson_line[count:-1]
    
    #saves the text between the opening and closing tag
    for char in lesson_line:
        if char == "<":
            break
        today = today + char  
    
    #checks the day and tells it how many lessons there are on that day to look for
    if "Wednesday" in today:
        limit = 4
    elif ("Friday") in today:
        limit = 6
    else:
        limit = 5
    Daycount = 1

    #loop to find the lesson time and subject
    while Daycount <= limit:
        id_index = lesson_line.index('data-id="')
        lesson_line = lesson_line[id_index+9:]
        ID = ""
        for char in lesson_line:
            if char == '"':
                break
            else:
                ID = ID + str(char)
        text = daymap_get(f"https://daymap.gihs.sa.edu.au/DayMap/Student/plans/class.aspx?eid={ID}", username, password)
        #lesson_info.write(text)
        #lesson time class finding
        time_index = lesson_line.index("class='t'")
        count = 0

        #cuts down to minimise searching
        lesson_line = lesson_line[time_index+9:]

        #finds the end of the opening tag
        for char in lesson_line:
            count +=1
            if char == ">":
                break
        
        #cuts down to minimise searching
        lesson_line = lesson_line[count:]
        time = ""

        #takes the text between the tags for the time
        for char in lesson_line:
            
            if char == "<":
                break
            time = time + str(char)
        
        #finds the subject class
        subject_index = lesson_line.find("event")
        count = 0

        #cuts down to minimise searching
        lesson_line = lesson_line[subject_index+6:]

        #finds the end of the tag
        for char in lesson_line:
            count += 1
            if char == ">":
                break
        
        #cuts down to minimise searching
        lesson_line = lesson_line[count:]
        subject = ""

        #notes the subject name
        for char in lesson_line:
            if char == "<":
                break
            subject = subject + str(char)

        #adds it to the list of subjects
        lesson_list.append(subject[:-2])

        #adds it to a dictionary along with the date and time
        timetable[subject[:-2]] = [today, time, ID]
        Daycount += 1
        #if statement to check if the next school day is monday
    class_index = lesson_line.find("diaryDay")
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
        id_index = lesson_line.index('data-id="')
        lesson_line = lesson_line[id_index+9:]
        ID = ""
        for char in lesson_line:
            if char == '"':
                break
            else:
                ID = ID + str(char)
        time_index = lesson_line.index("class='t'")
        count = 0
        lesson_line = lesson_line[time_index+9:]
        for char in lesson_line:
            count +=1
            if char == ">":
                break
        lesson_line = lesson_line[count:]
        time = ""
        for char in lesson_line:
            
            if char == "<":
                break
            time = time + str(char)
        subject_index = lesson_line.find("event")
        count = 0
        lesson_line = lesson_line[subject_index+6:]
        for char in lesson_line:
            count += 1
            if char == ">":
                break
        lesson_line = lesson_line[count:]
        subject = ""
        for char in lesson_line:
            if char == "<":
                break
            subject = subject + str(char)
        lesson_list2.append(subject[:-2])
        tomorrow_timetable[subject[:-2]] = [tomorrow, time, ID]
        Daycount += 1
    return week, today, timetable, lesson_list, tomorrow_timetable, lesson_list2

#function to pull JSON data from daymap
def get_daymapID(username, password):
    
#gets the json text from daymap
    html_text = daymap_get("https://daymap.gihs.sa.edu.au/daymap/DWS/Diary.ashx?cmd=EventList&from={2021-15-9}&to={2021-16-9}", username, password)

#formats it like a proper json file
    html_text = "{\"lesson_data\":" + html_text + "}"

#opens the json file and the writes the text to it, then closes it
    data = open("./usr/lesson-id.json", "w")
    data.write(html_text)
    data.close()

#reopens for the json module to sort out the data
    with open("./lib/csv/lesson-id.json") as json_data:
        data = json.load(json_data)
    lesson_data = data["lesson_data"]
    sorted_data = open("./usr/lesson-id.csv", "w")
    csv_writer = csv.writer(sorted_data)
    count = 0

#sorts the data into csv format
    for subject in lesson_data:
        if count == 0:
            header = subject.keys()
            csv_writer.writerow(header)
            count += 1
        csv_writer.writerow(subject.values())
    sorted_data.close()

# Function to get the specified user's DayMap messages.
def get_msgs(username, password):
    msgs = {}
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

        while "message|" in perm_line and msg_count < 3:
            index = line.index("message|")
            line = line[index+8:]
            ID = ""
            for char in line:
                if char == "'":
                    break
                ID = ID + str(char)
            perm_line = line
            msg_html = daymap_get(f"https://daymap.gihs.sa.edu.au/daymap/coms/Message.aspx?ID={ID}&via=4", username, password)
            msg_html = msg_html.split("\n")
            for line in msg_html:
                if "LabelRG msgSentOn" in line:
                    break
                else:
                    None
            index = line.index("LabelRG msgSentOn")
            line = line[index:]
            date = ""
            count = 0
            for char in line:
                count += 1
                if char == ">":
                    break 
            line = line[count:]
            for char in line:
                if char == "<":
                    break
                date = date + str(char)
            if "msgSubject" in line:
                index = line.index("msgSubject")
                line = line[index:]
                subject = ""
                count = 0
                for char in line:
                    count += 1
                    if char == ">":
                        break 
                line = line[count:]
                for char in line:
                    if char == "<":
                        break
                    subject = subject + str(char)
            else:
                subject = str(ID)
            
            index = line.index("msgSender")
            line = line[index:]
            sender = ""
            count = 0
            for char in line:
                count += 1
                if char == ">":
                    break 
            line = line[count:]    
            for char in line:
                if char == "<":
                    break
                sender = sender + str(char) 
            index = line.index("msgBody")
            line = line[index:]
            body = ""
            count = 0
            for char in line:
                count += 1
                if char == ">":
                    break 
            line = line[count:]   
                
            for char in line:
                if char == "<":
                    break
                body = body + str(char)
            
            subject = urllib.parse.unquote(subject)
            body = urllib.parse.unquote(body)
            date = urllib.parse.unquote(date)
            sender = urllib.parse.unquote(sender)
            msgs[ID] = [date, body, sender, subject]
            line = perm_line
            msg_count += 1
    except:
        None
    return msgs

# Function to get the specified user's tasks from DayMap.
def get_tasks(username, password):
    tasks = {}

    #note that this code will cause the webpage to be slow, hence why there is a different section for this
    page_html = daymap_get("https://daymap.gihs.sa.edu.au/daymap/student/dayplan.aspx", username, password)
    #file = open("./lib/csv/web_html", "w")
    #file.write(page_html)
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
                        print("overdue")
                        overdue = "class = 'err-bg'"
                        index = line.index("#FF4E1F")
                    else:
                        print("Not overdue")
                        overdue = ""
                        index = line.index("#68739B")
                elif "#68739B" in line:
                    print("NOT OVERDUE")
                    overdue = ""
                    index = line.index("#68739B")
                elif "#FF4E1F" in line:
                    print("OVERDUE")
                    overdue = "class = 'err-bg'"
                    index = line.index("#FF4E1F")
                notif_type = "Task"
                line = line[index:]
                index = line.index("OpenTask")
                line = line[index+9:]
                ID = ""
                for char in line:
                    if char == ")":
                        break
                    else:
                        ID += str(char)
                index = line.index("class='cap'")
                line = line[index:]
                count = 0
                for char in line:
                    count += 1
                    if char == ">":
                        break
                    
                line = line[count:-1]
                
                subject = ""
                for char in line:
                    if char == "<":
                        break
                    subject = subject+ str(char)
                count = 0
                for char in line:
                    count += 1
                    if char == ">":
                        break
                line = line[count:]
                count = 0
                for char in line:
                    count += 1
                    if char == ">":
                        break
                line = line[count:]
                
                sender = ""
                for char in line:
                    if char == "<":
                        break
                    sender = sender + char
                line = line[len(sender)+6:]
                
                due = ""
                for char in line:
                    if char == "<":
                        break
                    due = due + str(char)
                
                index = line.index("Caption")
                line = line[index:]
                count = 0
                for char in line:
                    count += 1
                    if char == ">":
                        break
                line = line[count:]
                
                assessment_type = ""
                for char in line:
                    if char == "<":
                        break
                    assessment_type = assessment_type + str(char)
                line = line[len(assessment_type)+11:]
                
                task_name = ""
                for char in line:
                    if char == "<":
                        break
                    task_name = task_name + str(char)
                
                tasks[task_name] = [subject, sender, due, assessment_type, overdue, notif_type, ID, "daymap-msgbox"]
    print(tasks)
    return tasks

    

    
