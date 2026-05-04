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

func TestNewJavaReferencedAnnotated(t *testing.T) {
	c := NewJavaReferencedAnnotated("javax.ejb.Stateless", LocationAnnotation, &Annotated{
		Pattern:  "javax.ejb.TransactionAttribute",
		Elements: []AnnotatedElement{{Name: "value", Value: "REQUIRED"}},
	})
	if c.JavaReferenced.Annotated == nil {
		t.Fatal("Annotated is nil")
	}
	if c.JavaReferenced.Annotated.Pattern != "javax.ejb.TransactionAttribute" {
		t.Errorf("annotated pattern: got %q", c.JavaReferenced.Annotated.Pattern)
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

func TestNewBuiltinFile(t *testing.T) {
	c := NewBuiltinFile("persistence.xml")
	if c.BuiltinFile == nil {
		t.Fatal("BuiltinFile is nil")
	}
}

func TestNewBuiltinXML(t *testing.T) {
	c := NewBuiltinXML("//beans:bean", map[string]string{"beans": "http://www.springframework.org/schema/beans"})
	if c.BuiltinXML == nil {
		t.Fatal("BuiltinXML is nil")
	}
}

func TestNewBuiltinJSON(t *testing.T) {
	c := NewBuiltinJSON("$.dependencies")
	if c.BuiltinJSON == nil {
		t.Fatal("BuiltinJSON is nil")
	}
}

func TestNewBuiltinHasTags(t *testing.T) {
	c := NewBuiltinHasTags([]string{"EJB"})
	if c.BuiltinHasTags == nil {
		t.Fatal("BuiltinHasTags is nil")
	}
}

func TestNewBuiltinXMLPublicID(t *testing.T) {
	c := NewBuiltinXMLPublicID(`-//Sun Microsystems.*`, nil)
	if c.BuiltinXMLPublicID == nil {
		t.Fatal("BuiltinXMLPublicID is nil")
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

func TestNewAnd(t *testing.T) {
	c := NewAnd(
		NewJavaDependency("javax.jms:javax.jms-api", "0.0.0", ""),
		NewJavaReferenced("javax.jms.MessageListener", LocationImplementsType),
	)
	if len(c.And) != 2 {
		t.Fatalf("expected 2 and entries, got %d", len(c.And))
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
