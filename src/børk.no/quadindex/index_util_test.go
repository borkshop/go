package quadindex

func (qi *Index) Data(i int) (int, Key) {
	return qi.ix[i], qi.ks[i]
}
