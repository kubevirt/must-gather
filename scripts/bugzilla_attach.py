#!/bin/python3

"""
Parameters:
    
    id: The Bugzilla bug ID to send the attachment to
    image: The image to use (defaults to quay.io/kubevirt/must-gather)
    api-key: Use a generated API key instead of a username and login
    log-folder: Use a specific folder for storing the output of must-gather

Requirements:

    Openshift 4.1+
    Python 3.6+
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
import itertools
import subprocess
import tarfile
import datetime
import base64
from getpass import getpass
import requests

# 100,000 lines gives a pre-compressed size of ~40MB and a compressed size of ~4MB
# Without trimming large files, the compressed output can exceed 40MB
MAX_LOGLINES = 100000

BUGZILLA_URL = "https://bugzilla.redhat.com"

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
                        help="Optional API key instead of username and login (will disable prompts to retry). Can also be set using BUGZILLA_API_KEY environment variable")
    parser.add_argument("--log-folder", metavar="log-folder",
                        help="Optional destination for the must-gather output (defaults to creating gather-files/ in the local directory)")
    parser.add_argument("-r", "--reuse-must-gather", action="store_true",
                        help="Use this to skip rerunning must-gather and just attach what is already gathered")
    parser.add_argument("-i", "--interactive", action="store_true",
                        help="Use this flag to prompt for a username and password")
    args = parser.parse_args()

    bug_id = args.ID

    if not check_bug_exists(bug_id):
        print("Bug not found in Bugzilla")
        exit(1)

    # If an image is supplied, use that, if not, use the default
    if args.image:
        image = args.image
    else:
        image = IMAGE

    # If a folder is supplied, use that, otherwise use the default in the local folder
    if args.log_folder:
        logfolder = args.log_folder
    else:
        logfolder = LOGFOLDER

    api_key = os.environ.get('BUGZILLA_API_KEY', "")

    if args.api_key:
        api_key = args.api_key

    # If there is no API key provided, prompt for a login
    use_api_key = api_key != None and api_key != ""
    if not use_api_key:
        if args.interactive:
            bugzilla_username = input("Enter Bugzilla username: ")
            bugzilla_password = getpass(prompt="Enter Bugzilla password: ")
        else:
            print("No API key supplied and not in interactive mode.")
            exit(1)

    if not args.reuse_must_gather:
        run_must_gather(image, logfolder)
    else:
        print("Using must-gather results located in %s." % logfolder)

    # Recursively walk the log folder and trim large files
    find_trim_files(logfolder)

    # Create a time-stamped archive name
    archive_name = ARCHIVE_NAME + "-%s.tar.gz" % datetime.datetime.now().strftime("%Y-%m-%d_%H-%M-%S")

    # Add all the files in the log folder to a new archive file, except for the hidden ones
    with tarfile.open(archive_name, "w:gz") as tar:
        print("Creating archive: " + archive_name)
        tar.add(logfolder, arcname=os.path.basename(logfolder), filter=filter_hidden)

    # Now that the archive is created, move the files back in place of the trimmed versions
    for subdir, _, files in os.walk(logfolder):
        for file in files:
            # If the file is hidden, it was a trimmed file so restore it
            if file[0] == ".":
                shutil.move(subdir + file, subdir + file[1:])


    print("Preparing to send the data to " + BUGZILLA_URL)

    file_data = ""
    with open(archive_name, "rb") as data_file:
        file_data = base64.b64encode(data_file.read()).decode()

    comment = generate_comment(image)

    # Send the data to the target URL (depending on whether using API key or not)
    if use_api_key:
        authentication = {"api_key": api_key}
    else:
        authentication = {"username": bugzilla_username, "password:": bugzilla_password}
    resp = send_data(bug_id, archive_name, file_data, comment, authentication)
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
                exit(0)
            bugzilla_password = getpass(prompt="Password: ")
            authentication = {"username": bugzilla_username, "password:": bugzilla_password}
            resp = send_data(bug_id, archive_name, file_data, comment, authentication)
            resp_json = resp.json()
        # 101: Invalid bug id
        elif resp_json["code"] == 101:
            print("Invalid bug id")
            new_bug_id = input("Enter a new bug id (leave blank to exit): ")
            if new_bug_id == "":
                print("ID left blank, exiting")
                exit(0)
            bug_id, valid = try_parse_int(new_bug_id)
            # Try and see if the new supplied ID is a positive integer
            while not valid or bug_id <= 0:
                print("Could not parse bug id as valid, try again")
                new_bug_id = input("Enter a new bug id (leave blank to exit): ")
                if new_bug_id == "":
                    print("ID left blank, exiting")
                    exit(0)
                bug_id, valid = try_parse_int(new_bug_id)
            resp = send_data(bug_id, archive_name, file_data, comment, authentication)
            resp_json = resp.json()
        else:
            print("Error: " + resp_json["message"])
            exit(1)
    print("File successfully uploaded to Bugzilla")

def run_must_gather(image, logfolder):
    # If the log folder already exists, delete it
    if os.path.isdir(logfolder):
        shutil.rmtree(logfolder)

    # Make a new log folder
    os.mkdir(logfolder)

    # Open the output file
    with open(logfolder + OUTPUT_FILE, "w+") as out_file:
        # Run oc adm must-gather with the appropriate image and dest-dir
        print("Running must-gather")
        try:
            subprocess.run(
                ["oc", "adm", "must-gather",
                "--image=" + image, "--dest-dir=" + logfolder],
                stdout=out_file, check=True)
        except subprocess.CalledProcessError:
            exit(1)


def find_trim_files(logfolder):
    # Recursively walk the log folder
    print("Searching for files to trim")
    for subdir, _, files in os.walk(logfolder):
        for file in files:
            file_name = os.path.join(subdir, file)
            with open(file_name, "r+") as curr_file:
                # Check the number of lines in each file
                # Even after compression, files that are too long will cause the attachment to exceed 19.5MB
                num_lines = get_lines(curr_file)
                if num_lines > MAX_LOGLINES:
                    # If the maximum number of lines is too high, trim it
                    # Copy it so that the original can be replaced
                    shutil.copyfile(file_name, os.path.join(subdir, "." + file))
                    print("Trimming %s because it exceeds %d lines"
                        % (os.path.join(subdir, file), MAX_LOGLINES))
                    trim_file(curr_file, MAX_LOGLINES)


def try_parse_int(value):
    """Tries to parse the value as an int"""
    try:
        return int(value), True
    except ValueError:
        return value, False



def send_data(bug_id, file_name, file_data, comment, authentication):
    url = BUGZILLA_URL + '/rest/bug/%s/attachment' % bug_id
    data = {
        **authentication,
        "ids": [bug_id],
        "comment": comment,
        "summary": "Result from must-gather command",
        "content_type": "application/gzip",
        "file_name": file_name,
        "data": file_data
    }
    return requests.post(url, json=data, headers=HEADERS)


def check_bug_exists(bug_id):
    """Checks whether the bug exists in Bugzilla"""
    url = BUGZILLA_URL + '/rest/bug/%s' % bug_id
    return "error" not in requests.get(url).json()

def get_lines(file):
    """Gets the number of lines in the file handle"""
    return len(file.readlines())

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

def generate_comment(image):
    """Creates the comment text for the attachment"""
    comment = ""
    comment += "Result from running oc adm must-gather --image=%s\n" % image
    comment += "Any file that exceeded %s lines was trimmed in order to reduce the size of the attachment\n" % MAX_LOGLINES
    return comment

def filter_hidden(filename):
    """Filters out hidden files so that the untrimmed ones won't be added to the archive"""
    return filename if os.path.split(filename.name)[1][0] != "." else None

main()
