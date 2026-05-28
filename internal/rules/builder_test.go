package rules

import "testing"

func TestNewJavaReferenced(t *testing.T) {
	c := NewJavaReferenced("javax.ejb.Stateless", LocationAnnotation)
	if c.JavaReferenced == nil {
		t.Fatal("JavaReferenced is nil")
	}
	if c.JavaReferenced.Pattern != "javax.ejb.Stateless" {
		t.Errorf("pattern: got %q", c.JavaReferenced.Pattern)
	}
	if c.JavaReferenced.Location != LocationAnnotation {
		t.Errorf("location: got %q", c.JavaReferenced.Location)
	}
}

func TestNewJavaDependency(t *testing.T) {
	c := NewJavaDependency("javax.jms:javax.jms-api", "1.0.0", "2.0.0")
	if c.JavaDependency == nil {
		t.Fatal("JavaDependency is nil")
	}
	if c.JavaDependency.Lowerbound != "1.0.0" {
		t.Errorf("lowerbound: got %q", c.JavaDependency.Lowerbound)
	}
}

func TestNewGoReferenced(t *testing.T) {
	c := NewGoReferenced("github.com/old/pkg.Function")
	if c.GoReferenced == nil {
		t.Fatal("GoReferenced is nil")
	}
}

func TestNewGoDependency(t *testing.T) {
	c := NewGoDependency("github.com/old/pkg", "1.0.0", "")
	if c.GoDependency == nil {
		t.Fatal("GoDependency is nil")
	}
}

func TestNewNodejsReferenced(t *testing.T) {
	c := NewNodejsReferenced("express.Router")
	if c.NodejsReferenced == nil {
		t.Fatal("NodejsReferenced is nil")
	}
}

func TestNewCSharpReferenced(t *testing.T) {
	c := NewCSharpReferenced("System.Web.HttpContext", CSharpLocationClass)
	if c.CSharpReferenced == nil {
		t.Fatal("CSharpReferenced is nil")
	}
	if c.CSharpReferenced.Location != CSharpLocationClass {
		t.Errorf("location: got %q", c.CSharpReferenced.Location)
	}
}

func TestNewPythonReferenced(t *testing.T) {
	c := NewPythonReferenced("flask.Flask")
	if c.PythonReferenced == nil {
		t.Fatal("PythonReferenced is nil")
	}
	if c.PythonReferenced.Pattern != "flask.Flask" {
		t.Errorf("pattern: got %q", c.PythonReferenced.Pattern)
	}
}

func TestNewBuiltinFilecontent(t *testing.T) {
	c := NewBuiltinFilecontent("spring\\.datasource", `application.*\.properties`)
	if c.BuiltinFilecontent == nil {
		t.Fatal("BuiltinFilecontent is nil")
	}
}

func TestNewBuiltinXML(t *testing.T) {
	c := NewBuiltinXML("//beans:bean", map[string]string{"beans": "http://www.springframework.org/schema/beans"})
	if c.BuiltinXML == nil {
		t.Fatal("BuiltinXML is nil")
	}
}

func TestNewOr(t *testing.T) {
	c := NewOr(
		NewJavaReferenced("javax.ejb.Stateless", LocationAnnotation),
		NewJavaReferenced("javax.ejb.Stateful", LocationAnnotation),
	)
	if len(c.Or) != 2 {
		t.Fatalf("expected 2 or entries, got %d", len(c.Or))
	}
}

func TestCondition_ChainingFields(t *testing.T) {
	c := NewJavaReferenced("javax.ejb.Stateless", LocationAnnotation).
		WithAs("ejb-check").
		WithNot(true).
		WithIgnore(false)

	if c.As != "ejb-check" {
		t.Errorf("as: got %q", c.As)
	}
	if c.Not == nil || *c.Not != true {
		t.Error("not: expected true")
	}
	if c.Ignore == nil || *c.Ignore != false {
		t.Error("ignore: expected false")
	}

	c2 := NewGoReferenced("pkg.Func").WithFrom("ejb-check")
	if c2.From != "ejb-check" {
		t.Errorf("from: got %q", c2.From)
	}
}
