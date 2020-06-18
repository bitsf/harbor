*** Settings ***
Documentation  Harbor BATs
Resource  ../../resources/APITest-Util.robot
Resource  ../../resources/Docker-Util.robot
Library  ../../apitests/python/library/Harbor.py  ${SERVER_CONFIG}
Library  OperatingSystem
Library  String
Library  Collections
Library  requests
Library  Process
Default Tags  APIDB

*** Variables ***
${SERVER}  ${ip}
${SERVER_URL}  https://${SERVER}
${SERVER_API_ENDPOINT}  ${SERVER_URL}/api
&{SERVER_CONFIG}  endpoint=${SERVER_API_ENDPOINT}  verify_ssl=False

*** Test Cases ***
Test Case - Scan Image
    [Tags]  scan
    Harbor API Test  ./tests/apitests/python/test_scan_image_artifact.py
Test Case - Scan All Images
    [Tags]  scan_all
    Harbor API Test  ./tests/apitests/python/test_system_level_scan_all.py
Test Case - Registry API
    [Tags]  reg_api
    Harbor API Test  ./tests/apitests/python/test_registry_api.py
Test Case - Push Image With Special Name
    [Tags]  special_repo_name
    Harbor API Test  ./tests/apitests/python/test_push_image_with_special_name.py
Test Case - Push Artifact With ORAS CLI
    [Tags]  oras
    Harbor API Test  ./tests/apitests/python/test_push_files_by_oras.py
Test Case - Push Singularity file With Singularity CLI
    [Tags]  singularity
    Harbor API Test  ./tests/apitests/python/test_push_sif_by_singularity.py
Test Case - Push Chart File To Chart Repository By Helm V2 With Robot Account
    [Tags]  helm2
    Harbor API Test  ./tests/apitests/python/test_push_chart_by_helm2_helm3_with_robot_Account.py
