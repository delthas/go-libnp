libnp:
	rm -rf libnp
	git clone https://github.com/delthas/libnp.git
	rm -fr libnp/.git
	git add libnp

.PHONY: libnp
