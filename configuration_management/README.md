ansible-playbook -i inventory.ini site.yml

ansible-playbook -i configuration_management/inventory.ini configuration_management/site.yml -e "target=myhosts"
                                                                                                          
To do a dry-run

ansible-playbook -i configuration_management/inventory.ini configuration_management/site.yml -e "target=myhosts" --check

###

ansible-playbook -i configuration_management/inventory.ini configuration_management/deploy.yml -e "target=myhosts" -e "repo_url=https://github.com/kennethatria/myGuy.git" 

### deploy certain branches

ansible-playbook -i configuration_management/inventory.ini configuration_management/deploy.yml -e "target=myhosts" -e "repo_url=https://github.com/kennethatria/myGuy.git" -e "git_branch=infrav2"