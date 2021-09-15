# Required for file existence checks.
import os.path

# Provides easy-to-use Google-supported Google Classroom data collection functionality.
from googleapiclient.discovery import build

# Required for a Google-supported front-end user authentication process.
from google_auth_oauthlib.flow import InstalledAppFlow

# Required for certain Google-supported update requests.
from google.auth.transport.requests import Request

# Provides Google-supported OAuth2 authentication to Google Classroom.
from google.oauth2.credentials import Credentials

# Function to get the specified user's tasks from Google Classroom.
def get_tasks(username, password):

    creds = None
    usrid = username[username.index("\\")+1:]

    # If modifying these scopes, delete the file token.json.
    # This development version has all related scopes enabled;
    # strip to the absolute necessary ones in the final version.

    SCOPES = [
            "https://www.googleapis.com/auth/classroom.courses.readonly",
            "https://www.googleapis.com/auth/classroom.course-work.readonly",
            "https://www.googleapis.com/auth/classroom.student-submissions.me.readonly",
            "https://www.googleapis.com/auth/classroom.coursework.me",
            "https://www.googleapis.com/auth/classroom.coursework.me",
            "https://www.googleapis.com/auth/classroom.courseworkmaterials.readonly",
            "https://www.googleapis.com/auth/classroom.topics.readonly"
            ]

    # Checks if user credentials are available.
    if os.path.isfile(f'./usr/{usrid}/classroom-creds.json'):
        creds = Credentials.from_authorized_user_file(
           f'./usr/{usrid}/classroom-creds.json', SCOPES
        )

    """
    I haven't yet found how to authenticate with a username and password;
    this seems to be the 'Google' way:
    https://developers.google.com/classroom/quickstart/python
    """

    # If there are no valid credentials available, let the user log in.
    if not creds or not creds.valid:

        # If user credentials have expired, send a refresh request.
        if creds and creds.expired and creds.refresh_token:
            creds.refresh(Request())

        # If there are no credentials, setup an authentication webpage.
        else:
            flow = InstalledAppFlow.from_client_secrets_file(
                './usr/classroom-token.json', SCOPES)
            creds = flow.run_local_server(port=14701)

        if not os.path.isdir(f"./usr/{usrid}"):
            os.mkdir(f"./usr/{usrid}")

        # Save the credentials.
        with open(f'./usr/{usrid}/classroom-creds.json', 'w') as newcreds:
            newcreds.write(creds.to_json())

    # Creates a Python object for getting Google Classroom data.
    service = build('classroom', 'v1', credentials=creds)

    # TODO: Collect tasks from 'service'.

    return {}