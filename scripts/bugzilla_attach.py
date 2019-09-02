#!/bin/python3

"""
Author: Avram Levitter

Parameters:
    
    id: The Bugzilla bug ID to send the attachment to
    image: The image to use (defaults to quay.io/kubevirt/must-gather)
    api-key: Use a generated API key instead of a username and login

Requirements:

    Openshift 4.1
    Python 3.6
    A Bugzilla account for www.bugzilla.redhat.com

This script attaches the result of the must-gather command, as executed
by the kubevirt must-gather image, to the supplied bugzilla id on the
Red Hat bugzilla website.
It first creates an output subdirectory in the working directory named
gather-files/ and then executes the following command:
'oc adm must-gather --image=quay.io/kubevirt/must-gather
--dest-dir=gather-files/' and pipes the output to
gather-files/must-gather.log
In order to meet the maximum attachment file sizes, it searches the
created directory for any log files that exceed 100,000 lines, and if
there are any, it trims those files down to 100,000 lines.
It then creates a time-stamped archive file to compress the attachment
and prepare it for upload.
After doing so, the attachment is encoded as a base64 string.
In order to authenticate against the Bugzilla website, a username and
password are prompted. A POST request is then sent to the Bugzilla
website. If there are any errors (invalid ID or invalid login), the
script prompts for those and retries the request until there are no
errors.
"""

import argparse
import os
import shutil
import subprocess
import tarfile
import datetime
import base64
from getpass import getpass
import requests

MAX_LOGLINES = 100000

BUGZILLA_URL = "https://www.bugzilla.redhat.com"

HEADERS = {'Content-type': 'application/json'}

LOGFOLDER = "gather-files/"

OUTPUT_FILE = "must-gather.log"

ARCHIVE_NAME = "must-gather"

IMAGE = "quay.io/kubevirt/must-gather"

def main():
    """Main function"""

    # Start with getting command-line argument(s)
    parser = argparse.ArgumentParser(description="Sends the result of must-gather to Bugzilla.")
    parser.add_argument("ID", metavar="id", type=int,
                        help="The ID of the bug in Bugzilla")
    parser.add_argument("--image", metavar="image",
                        help="The image to use for must-gather")
    parser.add_argument("--api-key", metavar="api-key",
                        help="Optional API key instead of username and login (will disable prompts to retry)")
    args = parser.parse_args()

    bug_id = args.ID

    if args.image:
        image = args.image
    else:
        image = IMAGE

    use_api_key = args.api_key != None

    # If the log folder already exists, delete it
    if os.path.isdir(LOGFOLDER):
        shutil.rmtree(LOGFOLDER)

    # Make a new log folder
    os.mkdir(LOGFOLDER)

    # Open the output file
    with open(LOGFOLDER + OUTPUT_FILE, "w+") as out_file:
        # Run oc adm must-gather with the appropriate image and dest-dir
        print("Running must-gather")
        subprocess.run(
            ["oc", "adm", "must-gather",
            "--image=" + image, "--dest-dir=" + LOGFOLDER],
            stdout=out_file)
    
    # Recursively walk the log folder
    print("Searching for files to trim")
    for subdir, _, files in os.walk(LOGFOLDER):
        for file in files:
            file_name = os.path.join(subdir, file)
            with open(file_name, "r+") as curr_file:
                # Check the number of lines in each file
                num_lines = get_lines(curr_file)
                if num_lines > MAX_LOGLINES:
                    # If the maximum number of lines is too high, trim it
                    print("Trimming %s because it exceeds %d lines"
                        % (os.path.join(subdir, file), MAX_LOGLINES))
                    trim_file(curr_file, MAX_LOGLINES)

    # Create a time-stamped archive name
    archive_name = ARCHIVE_NAME + "%s.tar.gz" % datetime.datetime.now().strftime("Y-%m-%d_%H-%M-%S")

    # Add all the files in the log folder to a new archive file
    with tarfile.open(archive_name, "w:gz") as tar:
        print("Creating archive: " + archive_name)
        tar.add(LOGFOLDER, arcname=os.path.basename(LOGFOLDER))

    print("Preparing to send the data to " + BUGZILLA_URL)

    file_data = ""
    with open(archive_name, "rb") as data_file:
        file_data = base64.b64encode(data_file.read())


    # Send the data to the target URL (depending on whether using API key or not)
    if use_api_key:
        resp = send_data_with_api_key(args.api_key, bug_id, file_data)
    else:
        bugzilla_username = input("Enter Bugzilla username: ")
        bugzilla_password = getpass(prompt="Enter Bugzilla password: ")
        resp = send_data(bugzilla_username, bugzilla_password, bug_id, file_data)
    resp_json = resp.json()

    # Handle the potential errors
    while "error" in resp_json:
        # Using an api key will disable retries, so just output the error message
        if use_api_key:
            print(resp_json["message"])
            break
        # 300: invalid username or password
        if resp_json["code"] == 300:
            print("Incorrect username or password.")
            bugzilla_username = input("Username (leave blank to exit): ")
            if bugzilla_username == "":
                print("Username left blank, exiting")
                exit()
            bugzilla_password = getpass(prompt="Password: ")
            resp = send_data(bugzilla_username, bugzilla_password, bug_id, file_data)
            resp_json = resp.json()
        # 101: Invalid bug id
        elif resp_json["code"] == 101:
            print("Invalid bug id")
            new_bug_id = input("Enter a new bug id (leave blank to exit): ")
            if new_bug_id == "":
                print("ID left blank, exiting")
                exit()
            bug_id, valid = try_parse_int(new_bug_id)
            # Try and see if the new supplied ID is a positive integer
            while not valid or bug_id <= 0:
                print("Could not parse bug id as valid, try again")
                new_bug_id = input("Enter a new bug id (leave blank to exit): ")
                if new_bug_id == "":
                    print("ID left blank, exiting")
                    exit()
                bug_id, valid = try_parse_int(new_bug_id)
            resp = send_data(bugzilla_username, bugzilla_password, bug_id, file_data)
            resp_json = resp.json()


def try_parse_int(value):
    """Tries to parse the value as an int"""
    try:
        return int(value), True
    except ValueError:
        return value, False

def send_data(username, password, bug_id, file_data):
    """Sends the data to the Bugzilla URL as an attachment"""
    url = BUGZILLA_URL + 'rest/bug/%s/attachment' % bug_id
    data = {
        "login": username,
        "password": password,
        "ids": [bug_id],
        "summary": "Result from must-gather command",
        "content_type": "application/gzip",
        "data": file_data
    }
    return requests.post(url, json=data, headers=HEADERS)

def send_data_with_api_key(api_key, bug_id, file_data):
    """Sends the data but uses an API key instead of a username and password"""
    url = BUGZILLA_URL + 'rest/bug/%s/attachment' % bug_id
    data = {
        "api_key": api_key,
        "ids": [bug_id],
        "summary": "Result from must-gather command",
        "content_type": "application/gzip",
        "data": file_data
    }
    return requests.post(url, json=data, headers=HEADERS)

def get_lines(file):
    """Gets the number of lines in the file handle"""
    return sum(1 for line in file)

def trim_file(file, num_lines):
    """Trims the file handle to the number of lines"""
    file.seek(0)
    lines = [line.rstrip('\n') for line in file]
    lines = lines[-num_lines:]
    file.seek(0)
    file.truncate(0)
    file.write("File trimmed to last %d lines\n" % num_lines)
    for line in lines:
        file.write("%s\n" % line)

main()
