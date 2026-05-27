package groundtruth

import "testing"

func TestParseCoord(t *testing.T) {
	tests := []struct {
		input   string
		want    MavenCoord
		wantErr bool
	}{
		{
			input: "org.apache.httpcomponents:httpclient:4.5.14",
			want:  MavenCoord{"org.apache.httpcomponents", "httpclient", "4.5.14"},
		},
		{
			input:   "invalid",
			wantErr: true,
		},
		{
			input:   "group:artifact",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCoord(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestMavenURL(t *testing.T) {
	c := MavenCoord{"org.apache.httpcomponents", "httpclient", "4.5.14"}
	want := "https://repo1.maven.org/maven2/org/apache/httpcomponents/httpclient/4.5.14/httpclient-4.5.14.jar"
	if got := c.MavenURL(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseJapicmpXMLData(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<japicmp>
  <classes>
    <class fullyQualifiedName="org.apache.http.HttpEntity" changeStatus="REMOVED" binaryCompatible="false" sourceCompatible="false">
    </class>
    <class fullyQualifiedName="org.apache.http.client.HttpClient" changeStatus="MODIFIED" binaryCompatible="false" sourceCompatible="false">
      <methods>
        <method name="execute" changeStatus="REMOVED"/>
        <method name="getParams" changeStatus="REMOVED"/>
        <method name="close" changeStatus="MODIFIED"/>
      </methods>
      <fields>
        <field name="MAX_REDIRECTS" changeStatus="REMOVED"/>
      </fields>
    </class>
    <class fullyQualifiedName="org.apache.hc.client5.http.HttpClient" changeStatus="NEW" binaryCompatible="true" sourceCompatible="true">
    </class>
  </classes>
</japicmp>`

	changes, err := parseJapicmpXMLData([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}

	if len(changes) != 5 {
		t.Fatalf("got %d changes, want 5", len(changes))
	}

	// class_removed
	if changes[0].ChangeKind != "class_removed" || changes[0].OldAPI != "org.apache.http.HttpEntity" {
		t.Errorf("change 0: got %+v", changes[0])
	}

	// method_removed: execute
	if changes[1].ChangeKind != "method_removed" || changes[1].OldAPI != "org.apache.http.client.HttpClient.execute" {
		t.Errorf("change 1: got %+v", changes[1])
	}

	// method_removed: getParams
	if changes[2].ChangeKind != "method_removed" || changes[2].OldAPI != "org.apache.http.client.HttpClient.getParams" {
		t.Errorf("change 2: got %+v", changes[2])
	}

	// method_changed: close
	if changes[3].ChangeKind != "method_changed" || changes[3].OldAPI != "org.apache.http.client.HttpClient.close" {
		t.Errorf("change 3: got %+v", changes[3])
	}

	// field_removed
	if changes[4].ChangeKind != "field_removed" || changes[4].OldAPI != "org.apache.http.client.HttpClient.MAX_REDIRECTS" {
		t.Errorf("change 4: got %+v", changes[4])
	}
}

func TestParseJapicmpXMLDataMethodChanged(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<japicmp>
  <classes>
    <class fullyQualifiedName="org.apache.http.StatusLine" changeStatus="MODIFIED" binaryCompatible="false" sourceCompatible="false">
      <methods>
        <method name="getStatusCode" changeStatus="MODIFIED"/>
      </methods>
    </class>
  </classes>
</japicmp>`

	changes, err := parseJapicmpXMLData([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}

	if len(changes) != 1 {
		t.Fatalf("got %d changes, want 1", len(changes))
	}
	if changes[0].ChangeKind != "method_changed" {
		t.Errorf("got kind %q, want method_changed", changes[0].ChangeKind)
	}
}
