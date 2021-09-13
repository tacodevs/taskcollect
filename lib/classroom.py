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

    """
    THE IDEA (TO BE IMPLEMENTED):

      1. Import all Python modules recommended for easy Google Classrooom data
         collection.

      2. Provide two-stage authentication to Google Classroom using the Google
         Cloud Project token and user credentials.

      3. Get a list of active classes.

      4. For each class, get a list of assignments.

      5. Parse all assignment data and put parsed/stripped data into a
         dictionary, then return.

    PROBLEMS:
    Getting data from Google Classroom is expensive, as such it needs to be
    optimised.
    """

    return {}