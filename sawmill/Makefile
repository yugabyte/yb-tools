# $< first pre-req
# $^ all pre-reqs

sawmill: lex.yy.c
	cc $^ -ll -o sawmill

test: sawmill
	./$< foo.log bar.log

lex.yy.c: sawmill.l
	flex $<

clean:
	rm -f *~

clobber: clean
	rm -f lex.yy.c sawmill.tab.c sawmill.tab.h a.out sawmill
