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
    usrid = username[7:]

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
    # BUG: This frequently fails.
    # BUG: One bug is that the modules don't create a "refresh-token" field.
    # BUG: Another that I sometimes get are SSL certificate errors.
    if os.path.isfile(f'./usr/{usrid}/classroom-creds.json'):
        creds = Credentials.from_authorized_user_file(
           f'./usr/{usrid}/classroom-creds.json', SCOPES
        )

    """
    I haven't yet found how to authenticate with a username and password;
    this seems to be the 'Google' way:
    https://developers.google.com/classroom/quickstart/python

    Refer to this for increasing Google Classroom performance:
    https://developers.google.com/classroom/guides/performance
    We definitely need to work with partial resources.
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
            # BUG: NOTHING should be run locally, so 'run_local_server' does not make sense.
            # Perhaps try to get authentication working through a username and password.
            creds = flow.run_local_server(port=14701)

        if not os.path.isdir(f"./usr/{usrid}"):
            os.mkdir(f"./usr/{usrid}")

        # Save the credentials.
        with open(f'./usr/{usrid}/classroom-creds.json', 'w') as newcreds:
            newcreds.write(creds.to_json())

    # Creates a Python object for getting Google Classroom data.
    service = build('classroom', 'v1', credentials=creds)

    # Gets a Python dictionary of all Google Classroom subjects.
    subjects = service.courses().list().execute().get('courses', [])

    tasks = []
    n = 0

    # Retrieve assignments for all active subjects, and get all necessary data.
    # BUG: Currently gets data for all subjects, including archived subjects.
    for subject in subjects:

        assignments = service.courses().courseWork().list(courseId=subject['id']).execute()

        # Puts the necessary data into 'tasks' in the desired format.
        if 'courseWork' in assignments:
            for assignment in assignments['courseWork']:

                tasks.append([])
                tasks[n].append(assignment['title'])
                tasks[n].append(subject['name'])

                try:
                    tasks[n].append(assignment['description'])
                except KeyError:
                    tasks[n].append("No description.")

                tasks[n].append(assignment['alternateLink'])

                try:
                    tasks[n].append(assignment['dueDate'])
                except:
                    tasks[n].append("No due date.")

                try:
                    tasks[n].append(assignment['dueTime'])
                except:
                    tasks[n].append("No due time.")

                # BUG: Incomplete code.
                tasks[n].append('overduestatus')
                tasks[n].append('Google Classroom')

                n += 1

    return tasks