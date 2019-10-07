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
import re
import subprocess
import tarfile
import datetime
import base64
from getpass import getpass
import requests

NUM_SECONDS = 30 * 60 # 30 minutes

BUGZILLA_URL = "https://bugzilla.redhat.com"

HEADERS = {'Content-type': 'application/json'}

LOGFOLDER = "gather-files/"

OUTPUT_FILE = "must-gather.log"

ARCHIVE_NAME = "must-gather"

IMAGE = "quay.io/kubevirt/must-gather"

NODELOG_TIMESTAMP_REGEX = re.compile(r"(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d+ \d+:\d+:\d+")

NODELOG_TIMESTAMP_FORMAT = "%b %d %H:%M:%S"

PODLOG_TIMESTAMP_REGEX = re.compile(r"^\d{4}-\d{2}-\d{2}T\d+:\d+:\d+")

PODLOG_TIMESTAMP_FORMAT = "%Y-%m-%dT%H:%M:%S"

_current_time = datetime.datetime.utcnow()

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
    parser.add_argument("-t", "--time", type=int, help="Number of seconds to use for trimming the log files. Defaults to 30 minutes.")
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

    if args.time:
        num_seconds = args.time
    else:
        num_seconds = NUM_SECONDS

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


    trim_logs(logfolder, num_seconds)
    # Trim the node logs and the pod logs
    #trim_node_logs(logfolder, num_seconds)
    #trim_pod_logs(logfolder, num_seconds)

    # Create a time-stamped archive name
    archive_name = ARCHIVE_NAME + "-%s.tar.gz" % _current_time.strftime("%Y-%m-%d_%H:%M:%SZ")

    # Add all the files in the log folder to a new archive file, except for the hidden ones
    with tarfile.open(archive_name, "w:gz") as tar:
        print("Creating archive: " + archive_name)
        tar.add(logfolder,
        #arcname=os.path.basename(logfolder),
        #arcname="",
        filter=filter_hidden)

    # Now that the archive is created, move the files back in place of the trimmed versions
    restore_hidden_files(logfolder)


    print("Preparing to send the data to " + BUGZILLA_URL)

    file_data = ""
    with open(archive_name, "rb") as data_file:
        file_data = base64.b64encode(data_file.read()).decode()

    comment = generate_comment(image, num_seconds)

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
            exit(1)
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
            print("Error in the execution of must-gather:")
            exit(1)


def trim_logs(logfolder, num_seconds):
    for subdir, _, files in os.walk(logfolder):
        for file in files:
            if file == "must-gather.log": #Ignore the log made by capturing the output of must-gather
                continue
            if ".log" in file:
                trim_file_by_time(os.path.join(subdir, file), num_seconds, PODLOG_TIMESTAMP_REGEX, PODLOG_TIMESTAMP_FORMAT)
            if "kubelet" in file or "NetworkManager" in file:
                trim_file_by_time(os.path.join(subdir, file), num_seconds, NODELOG_TIMESTAMP_REGEX, NODELOG_TIMESTAMP_FORMAT)

def trim_node_logs(logfolder, num_seconds):
    print("Trimming node logs")
    for node_name in os.listdir(logfolder + "nodes/"):
        trim_file_by_time(os.path.join(logfolder, "nodes", node_name, node_name + "_logs_kubelet"), num_seconds, NODELOG_TIMESTAMP_REGEX, NODELOG_TIMESTAMP_FORMAT)
        trim_file_by_time(os.path.join(logfolder, "nodes", node_name, node_name + "_logs_NetworkManager"), num_seconds, NODELOG_TIMESTAMP_REGEX, NODELOG_TIMESTAMP_FORMAT)

def trim_pod_logs(logfolder, num_seconds):
    print("Trimming pod logs")
    for namespace in os.listdir(logfolder + "namespaces/"):
        pod_folder = os.path.join(logfolder, "namespaces", namespace, "pods")
        if not os.path.exists(pod_folder):
            continue
        for subdir, _, files in os.walk(pod_folder):
            for file in files:
                if ".log" in file:
                    trim_file_by_time(os.path.join(subdir, file), num_seconds, PODLOG_TIMESTAMP_REGEX, PODLOG_TIMESTAMP_FORMAT)


def try_parse_int(value):
    """Tries to parse the value as an int"""
    try:
        return int(value), True
    except ValueError:
        return value, False



def send_data(bug_id, file_name, file_data, comment, authentication):
    """Sends the data to Bugzilla with the relevant information"""
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


def trim_file_by_time(filename, num_seconds, timestamp_regex, timestamp_format):
    """Uses a binary search to locate where in the file to trim"""
    with open(filename, "r+") as file:
        lines = [line.rstrip('\n') for line in file]
        num_lines = len(lines)
        # Perform a binary search over the lines to find where in the log file is n seconds ago
        current_index = round(num_lines / 2) # Start halfway through the file
        search_size = current_index # Search size is how much to move the index after checking the line
        while search_size > 1 and current_index > 1 and current_index < num_lines:
            line = lines[current_index]
            search_size = round(search_size / 2)
            # Match according to the designated search regex, then convert to a datetime object
            timestamp_string = timestamp_regex.match(line)[0]
            line_timestamp = datetime.datetime.strptime(timestamp_string, timestamp_format)
            # If the month of the line is greater than the current month, assume it's from last year
            if line_timestamp.month > _current_time.month:
                line_timestamp = line_timestamp.replace(year=_current_time.year() - 1)
            else:
                line_timestamp = line_timestamp.replace(year=_current_time.year)
            if (_current_time - line_timestamp).total_seconds() > num_seconds:
                #If the difference is larger than the max, go later in the file
                current_index += search_size
            else:
                #if not, go earler in the file
                current_index -= search_size
        if current_index == 1: # Meaning it reached near the beginning of the file
            return # Don't remove anything
        # Since this file will be trimmed, create a hidden copy of it
        hidden_filename = os.path.join(os.path.dirname(filename), "." + os.path.basename(filename))
        shutil.copy(filename, hidden_filename)
        file.seek(0)
        file.truncate(0)
        if current_index == num_lines: # Meaning there was nothing in the file that is to be kept
            pass
        # Slice from the current index to the end
        for line in lines[current_index:]:
            file.write("%s\n" % line)



def generate_comment(image, num_seconds):
    """Creates the comment text for the attachment"""
    comment = ""
    comment += "Result from running oc adm must-gather --image=%s\n" % image
    comment += "Log files were trimmed to the last %d" % num_seconds
    return comment

def filter_hidden(file):
    """Filters out hidden files so that the untrimmed ones won't be added to the archive"""
    return file if os.path.basename(os.path.normpath(file.name))[0] != "." else None

def restore_hidden_files(logfolder):
    """Finds any hidden files and renames them to their original name"""
    for subdir, _, files in os.walk(logfolder):
        for file in files:
            # If the file is hidden, it was a trimmed file so restore it
            if file[0] == ".":
                shutil.move(os.path.join(subdir, file), os.path.join(subdir, file[1:]))

main()
