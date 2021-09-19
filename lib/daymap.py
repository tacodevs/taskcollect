# Required for CSV parsing
import csv

# Required for easier date/time data manipulation
import datetime

# Required for JSON parsing
import json

# Required to send HTTP requests
import requests

# Required to make NTLM handshakes
import requests_ntlm

# HTML parser; JS already has one
from lxml import html

# Required to print to standard error output
import sys

# Required to parse HTML strings
import urllib.parse

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

# Function to get the lessons for a certain day.
def get_lessons(date, username, password):

    lessons = []

    # Get student landing webpage for DayMap.
    landpage = daymap_get(
        "https://daymap.gihs.sa.edu.au/daymap/student/dayplan.aspx",
        username, password
    )

    landpage = landpage.split("\n")

    # Get the line with actual lesson data.
    lessonline = landpage[211]
    
    # Looks for "week" class.
    week_index = lessonline.find("diaryWeek")

    # Cuts down the lesson line to minimise searching.
    lessonline = lessonline[week_index+10:-1]

    count = 0

    return lessons

# Function to get the specified user's DayMap messages.
def get_msgs(username, password):

    # TODO: Refactor.

    msgs = {}

    landpage = daymap_get(
        "https://daymap.gihs.sa.edu.au/daymap/student/dayplan.aspx",
        username, password
    )

    landpage = landpage.split("\n")

    for line in landpage:
        if "<div class='Header'>Messages </div>" in line:
            break
        else:
            landpage.remove(line)

    index = landpage.index(line)

    landpage = landpage[index:]

    for line in landpage:
        if "Messages" in line:
            break
        else:
            landpage.remove(line)

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

            msg_html = daymap_get(
                f"https://daymap.gihs.sa.edu.au/daymap/coms/Message.aspx?ID={ID}&via=4",
                username, password
            )

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
                subject = "<i>No title.</i>"
            
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

    tasks = []

    return tasks