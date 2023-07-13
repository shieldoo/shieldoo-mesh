#!/bin/bash

# The script needs to be run with elevated privileges to execute:
# apt-get update
# apt-get dist-upgrade
# apt-get install
# apt-get -f install
# dpkg --configure -a
#
# dnf upgrade

# Return values:
# 0	If no error occured
# 1	apt-get and dnf not found (by the "command" shell builtin) 
# 2	Wrong usage
# 3     A command that does not upgrade packages failed (apt-get update, apt-get dist-upgrade -s, dnf check-update)
# 4	A command that upgrades packages failed (apt-get dist-upgrade, apt-get install, dnf upgrade)
# 5	apt-check utility returns different number of security updates,
#	this probalby means that output format of 'apt-get -s -qq dist-upgrade' changed, which has broken this script

# ======================================================================================
# ===========================GENERAL USEFUL FUNCTIONS===================================

# Usage function
usage() {
  echo "Usage: $0 [option]"
  echo "Options:"
  echo "  -s, --security     Display available security updates"
  echo "  -o, --other        Display available non-security updates"
  echo "  -u, --update       Perform security update"
  echo "  -a, --all          Perform all updates (security and non-security)"
  echo "  -r, --recover      Try to recover if the upgrade process was interupted"
  echo "  -h, --help         Display this help message"
}

# Error message function
error() {
	printf 'Command "%s" failed with exit code %d\n' "$1" "$2" 
}

# ======================================================================================
# ======================================================================================


# ======================================================================================
# ===================FUNCTIONS USING APT-GET(FOR DEBIAN BASED DISTROS)==================

# Function to update available packages
apt_update_package_list() {
	#if apt-get update fails, echo the error messages and exit the script
	#else output nothing and continue the script
	output="$(apt-get -q update)"
	exit_code="$?"
	[[ "$exit_code" -ne 0 ]] && { echo "$output" >&2; error "apt-get update" "$exit_code" >&2; exit 3; }
}

# Function that finds security updates if coresponding argument is passed ("security"), else finds updates not related to security
apt_find_updates() {
	#if apt-get -s -qq dist-upgrade fails, echo the error message and exit the script
	#else continue the function
	output="$(apt-get -s -qq dist-upgrade)"
	exit_code="$?"
	[[ "$exit_code" -ne 0 ]] && { echo "$output" >&2; error "apt-get dist-upgrade -s" "$exit_code" >&2; exit 3; }
      	
	#check if positional parameter "security" is provided and set grep options accordingly
        #-i is for case-insensitive search
	#-v is for printing the content that does not match the search pattern (vice-versa) 	
	opt="-vi";
	if [[ "$1" == "security" ]]; then
		opt="-i"
		#reliably discover how many security updates are available using apt-check
		command -v /usr/lib/update-notifier/apt-check &>/dev/null && ref_count="$(/usr/lib/update-notifier/apt-check 2>&1 | cut -d';' -f2)"
	fi
	
	#list the security/other updates
	printf '%s\n' "$output" | grep ^Inst | grep "$opt" '\-security' | cut -d' ' -f2
	
	#check if the number of found security updates corresponds to the number returned by /usr/lib/update-notifier/apt-check
	#if not signal it via exit code
	#this check is needed because apt-get has no built-in filters for security updates and grep is dependent on the format of input
	[[ -v ref_count && "$ref_count" -ne "$(printf '%s\n' "$output" | grep ^Inst | grep "$opt" '\-security' | cut -d' ' -f2 | wc -l)" ]] && exit 5	
}

# Function to display security updates
apt_list_security_updates() {
	apt_update_package_list
	apt_find_updates "security"
}

# Function to display non-security updates
apt_list_other_updates() {
	apt_update_package_list
	apt_find_updates "other"
}

# Function to perform security update
apt_perform_security_updates() {
	apt_update_package_list

	#list security updates and install them without user interaction
	#quoting of the variable NONINTERACTIVE_OPTS is ommited intentionally to enable word splitting
	#xargs is used for executing apt-get install with the security packages as arguments	
	if ! apt_find_updates "security" | xargs apt-get $NONINTERACTIVE_OPTS install;
	then
		error "apt-get $NONINTERACTIVE_OPTS install" "$?" >&2
		exit 4
	fi
}

# Function to perform all updates
apt_perform_all_updates() {
	apt_update_package_list
	
	#upgrade all available packages
	#quoting of the variable containing options is ommited intentionally to enable word splitting	
	if ! apt-get $NONINTERACTIVE_OPTS dist-upgrade; 
	then
		error "apt-get $NONINTERACTIVE_OPTS dist-upgrade" "$?" >&2
		exit 4
	fi
}

# Function to recover from upgrade interrupt
# Note that the options for automated recovery are very limited, usually a manual repair is needed
apt_recover() {
	#attempt to correct a system with broken dependencies in place
	apt-get -f install
	
	#configure all packages that have been unpacked but not yet configured
	dpkg --configure -a
}

# ======================================================================================
# ======================================================================================

# ======================================================================================
# ======================FUNCTIONS USING DNF(FOR RHEL BASED DISTROS)=====================

# Function to display security/all updates based on parameter
dnf_find_updates() {
	#set options for dnf check-update based on positional parameter
	#-q is for omitting unnecessary information in the output
	#--security is for filtering security packages
	opt="-q"; [[ "$1" == "security" ]] && opt="-q --security"		

	#if dnf check-update fails (exit code 100 means no updates have been found),
	#echo the error messages and exit the script
	#else parse the output and continue the script
	output="$(dnf $opt check-update)"
	exit_code="$?"
	[[ "$exit_code" -ne 0 && "$exit_code" -ne 100 ]] && { echo "$output" >&2; error "dnf $opt check-update" "$exit_code" >&2; exit 3; }
	printf '%s\n' "$output" | cut -d' ' -f1 | grep -v '^$' | sed 's/\.[^\.]*$//'
}

# Function to display security updates
dnf_list_security_updates(){
	dnf_find_updates "security"
}

# Function to display non-security updates
dnf_list_other_updates() {
	#create temporary file
	tmpfile=$(mktemp) || { error "mktemp" "$?" >&2; exit 3; }
	
	#store the sorted list of all updates in the tmp file
	#sort is necessary for comm command to work
	dnf_find_updates "all" | sort  > "$tmpfile"
	
	#use comm to display the lines that appear only in the tmpfile,
	#thus are not security updates
	dnf_find_updates "security" | sort | comm -23 "$tmpfile" -
	
	#clean-up
	rm "$tmpfile"
}

# Function to perform security/all update depending on parameter
dnf_perform_update() {
	#perform non-interactive upgrade, if --security parameter is provided,
	#perform only security patches
	#quoting of the variable containing options is ommited intentionally to enable word splitting	
  	if ! dnf $NONINTERACTIVE_OPTS $1 upgrade;
	then
		#if dnf returns 1 it means an error occurred but dnf was able to solve it
		#in that case continue the script
		[[ "$?" -eq 1 ]] && return 0
		error "dnf $NONINTERACTIVE_OPTS $1 upgrade" "$?" >&2
		exit 4	
	fi
}

# Function to perform security updates
dnf_perform_security_updates(){
	dnf_perform_update "--security"
}

# Function to perform all updates
dnf_perform_all_updates(){
	dnf_perform_update ""
}

# Function to recover from upgrade interrupt
dnf_recover(){
	echo "No automated fixes are available, manual repair is needed."
}

# ======================================================================================
# ======================================================================================

# Check what package management software is present on the system and set corresponding variables.
if command -v apt-get &> /dev/null; then
    	SOFTWARE="apt"
	export DEBIAN_FRONTEND=noninteractive
	#NONINTERACTIVE_OPTS='-q -y -o Dpkg::Options::=--force-confold -o Dpkg::Options::=--force-confdef --allow-downgrades --allow-remove-essential --allow-change-held-packages'
	NONINTERACTIVE_OPTS='-q -y -o Dpkg::Options::=--force-confold -o Dpkg::Options::=--force-confdef'
elif command -v dnf &> /dev/null; then
	SOFTWARE="dnf"
	NONINTERACTIVE_OPTS='-y'
else	
	echo 'apt-get and dnf not found (by the "command" shell builtin)' 
	exit 1
fi	

# If one and only one option is not provided, display usage information
[[ $# -ne 1 ]] && { usage; exit 2; }

# Resolve command line option
# For simplicity only one option on each invocation is supported
  case $1 in
    -s|--security)
      # Display available security updates
      ${SOFTWARE}_list_security_updates
      exit 0
      ;;
    -o|--other)
      # Display available non-security updates
      ${SOFTWARE}_list_other_updates
      exit 0
      ;;
    -u|--update)
      # Perform security update
      ${SOFTWARE}_perform_security_updates
      exit 0
      ;;
    -a|--all)
      # Perform all updates (security and non-security)
      ${SOFTWARE}_perform_all_updates
      exit 0
      ;;
    -r|--recover)
      # Try to recover the packages if upgrade got interrupted
      ${SOFTWARE}_recover
      exit 0
      ;;
    -h|--help)
      # Display usage information
      usage
      exit 0
      ;;
    *)
      # Invalid option
      echo "Invalid option: $1" >&2
      usage
      exit 2
      ;;
  esac
