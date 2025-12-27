package main

import "fmt"

// 学生结构体
type Student struct {
	Id    int
	Name  string
	Age   int
	Grade int
	Class string
}

// 学生管理器
type StudentManager struct {
	students []Student
}

// 创建学生管理器
func CreateStudent() *StudentManager {
	return &StudentManager{
		// 创建一个学生空切片
		students: make([]Student, 0),
	}
}

// 添加学生
func (sm *StudentManager) AddStudent(student Student) error {
	//根据Id检查学生是否存在
	for _, s := range sm.students {
		if s.Id == student.Id {
			return fmt.Errorf("学生Id %d 已存在", student.Id)
		}
	}
	//把新学生添加到切片中
	sm.students = append(sm.students, student)
	return nil
}

// 删除学生
func (sm *StudentManager) DeleteStudent(id int) error {
	for i, student := range sm.students {
		if id == student.Id {
			//使用切片删除学生 把删除的元素后面的元素往前移动一位
			sm.students = append(sm.students[:i], sm.students[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("学生Id %d 不存在", id)
}

// 更新学生信息
func (sm *StudentManager) UpdateStudent(id int, updatedStudent Student) error {
	for i, student := range sm.students {
		if id == student.Id {
			updatedStudent.Id = id
			sm.students[i] = updatedStudent
			return nil
		}
	}
	return fmt.Errorf("学生Id %d 不存在", id)
}

// 根据Id查询学生
func (sm *StudentManager) GetStudent(id int) (Student, error) {
	var student Student
	for _, student := range sm.students {
		if id == student.Id {
			return student, nil
		}
	}
	return student, nil
}

// 根据条件查询学生
func (sm *StudentManager) FindStudents(name string, grade int) []Student {
	var students []Student
	for _, student := range sm.students {
		if (name == "" || student.Name == name) && (grade == 0 || student.Grade == grade) {
			students = append(students, student)
		}
	}
	return students
}

// 查询所有学生列表
func (sm *StudentManager) GetAllStudents() {
	fmt.Println("查询所有学生列表")
	if len(sm.students) == 0 {
		fmt.Println("没有学生")
	}
	for _, student := range sm.students {
		fmt.Printf("Id: %d, 姓名: %s, 年龄: %d, 分数: %d, 班级: %s\n", student.Id, student.Name, student.Age, student.Grade, student.Class)
	}
	fmt.Println("查询结束")
	fmt.Printf("总计: %d 位学生\n", len(sm.students))
}

func StudentManagementDemo() {
	sm := CreateStudent()
	sm.AddStudent(Student{Id: 1, Name: "张三", Age: 18, Grade: 90, Class: "1-1"})
	sm.AddStudent(Student{Id: 2, Name: "李四", Age: 17, Grade: 80, Class: "1-2"})
	sm.AddStudent(Student{Id: 3, Name: "王五", Age: 16, Grade: 70, Class: "1-3"})
	sm.AddStudent(Student{Id: 4, Name: "赵六", Age: 15, Grade: 60, Class: "1-4"})
	sm.AddStudent(Student{Id: 5, Name: "孙七", Age: 14, Grade: 50, Class: "1-5"})
	sm.AddStudent(Student{Id: 6, Name: "周八", Age: 13, Grade: 40, Class: "1-6"})
	sm.AddStudent(Student{Id: 7, Name: "吴九", Age: 12, Grade: 30, Class: "1-7"})
	sm.AddStudent(Student{Id: 8, Name: "郑十", Age: 11, Grade: 20, Class: "1-8"})

	sm.GetAllStudents()

	students := sm.FindStudents("张三", 0)
	for _, student := range students {
		fmt.Printf("Id: %d, 姓名: %s, 年龄: %d, 分数: %d, 班级: %s\n", student.Id, student.Name, student.Age, student.Grade, student.Class)
	}

	student, error := sm.GetStudent(2)
	if error != nil {
		fmt.Printf("Id: %d, 姓名: %s, 年龄: %d, 分数: %d, 班级: %s\n", student.Id, student.Name, student.Age, student.Grade, student.Class)
	}

	sm.UpdateStudent(2, Student{Id: 2, Name: "张三丰", Age: 19, Grade: 3, Class: "高三(1)班"})

	student1, _ := sm.GetStudent(2)
	fmt.Println("更新后的学生信息")
	fmt.Println("Id: %d, 姓名: %s, 年龄: %d, 分数: %d, 班级: %s\n", student1.Id, student1.Name, student1.Age, student1.Grade, student1.Class)

	fmt.Println("根据条件查询学生")
	seniorStudents := sm.FindStudents("", 70)
	for _, student := range seniorStudents {
		fmt.Printf("姓名: %s, 班级: %s\n", student.Name, student.Class)
	}

	//sm.DeleteStudent(2)

	sm.GetAllStudents()
}

func main() {
	StudentManagementDemo()
}
