package algo

func QuickSort[T comparable](s []T, cmp func(a, b T) int) {
	quickSort(s, cmp)
}

// quickSort 从左右逐步向中间移动，当从左边遇到大于分界值时尝试将其移动到右边，从右边同理。
// 最终当j>i不成立时停止移动。
// temp = s[p] 分界值
// 索引i从0递增，索引j从切片长度递减。p则根据情况取i或j的值。
// 当s[j]大于等于temp且j>p，索引j不断从右往左移动，当条件不满足时交换s[p]和s[j]，交换p和j。
// 当s[i]小于等于temp且i<p，索引i不断从左往右移动，当条件不满足时交换s[p]和s[i]，交换p和i。
// 结果：以索引p为分界线，左半部分的值都小于s[p]，右半部分都大于等于s[p]
func quickSort[T comparable](s []T, cmp func(a, b T) int) {
	l := len(s)
	// left := rand.Into(l)
	// s[0], s[left] = 0, s[0]
	i, j := 0, l-1
	p, temp := 0, s[0]

	for j > i {
		// for s[j] >= temp && j > p {
		for cmp(s[j], temp) >= 0 && j > p {
			j--
		}
		if j != p { // 当p等于j时，交换是没有意义的。
			s[p] = s[j] // 将s[j]放置到左半部分，因为其小于temp
			p = j
		}
		// 此时s[p]是一个小于或等于(上面的for循环因为j==p退出时)temp

		// for s[i] <= temp && i < p {
		for cmp(s[i], temp) <= 0 && i < p {
			i++
		}
		if i != p { // 当p等于i时，交换是没有意义的。
			s[p] = s[i] // 将s[i]放置到右半部分，因为其大于temp
			p = i
		}
		// 此时s[p]是一个大于或等于(上面的for循环因为i==p退出时)temp
	}
	s[p] = temp

	if p > 1 { // p-left > 1
		quickSort(s[:p], cmp)
	}
	if l-p > 2 { //  len(s)-1-p > 1
		quickSort(s[p+1:], cmp)
	}
}
