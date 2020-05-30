package audit

import (
	"regexp"

	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/klog"
)

func FilterEvent(event *audit.Event, userRegex *regexp.Regexp, UserFilterInverse bool, nsRegex *regexp.Regexp) bool {
	eventUser := &event.User
	if event.ImpersonatedUser != nil {
		eventUser = event.ImpersonatedUser
	}

	match := userRegex.MatchString(eventUser.Username)

	//  match    inverse
	//  -----------------
	//  true     true   --> skip
	//  true     false  --> keep
	//  false    true   --> keep
	//  false    false  --> skip
	if match {
		if UserFilterInverse {
			klog.V(5).Infof("skip %v", eventUser.Username)
			return false
		}
	} else {
		if !UserFilterInverse {
			klog.V(5).Infof("skip %v", eventUser.Username)
			return false
		}
	}

	if event.ObjectRef != nil && event.ObjectRef.Namespace != "" {
		match := nsRegex.MatchString(event.ObjectRef.Namespace)
		if !match {
			klog.V(5).Infof("skip namespace %v", event.ObjectRef.Namespace)
			return false
		}
	}

	return true
}
