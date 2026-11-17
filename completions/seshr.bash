_seshr() {
	local cur=${COMP_WORDS[COMP_CWORD]}
	if [[ $COMP_CWORD -eq 1 && upgrade == "$cur"* ]]; then
		COMPREPLY=(upgrade)
	fi
	if [[ ${COMP_WORDS[1]} == upgrade ]]; then
		COMPREPLY=($(compgen -W "-y --yes" -- "$cur"))
	else
		COMPREPLY+=($(compgen -W "--here --since --version" -- "$cur"))
	fi
}
complete -F _seshr seshr
