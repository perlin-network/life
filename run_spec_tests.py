import os
import sys
import subprocess

def collect_wast(dir):
    ret = []

    for item in os.listdir(dir):
        full_path = os.path.join(dir, item)
        if os.path.isfile(full_path) and full_path.endswith(".wast"):
            ret.append(full_path)

    return ret

wast_files = collect_wast(sys.argv[1])
success_list = []
failure_list = []
for name in wast_files:
    try:
        json_name = name + ".json"
        ret = subprocess.call(["wast2json", name, "-o", json_name ])
        if ret != 0:
            raise Exception("wast2json")
        ret = subprocess.call(["./test_runner", json_name])
        if ret != 0:
            raise Exception("test_runner")
        success_list.append(name)
    except Exception as e:
        print(e)
        failure_list.append(name)
print("Successes:")
print(success_list)
print("Failures:")
print(failure_list)

num_successes = len(success_list)
num_failures = len(failure_list)
print("{} successes, {} failures".format(num_successes, num_failures))
