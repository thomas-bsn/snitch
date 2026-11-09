import unittest

import check_engine_families as check


class EngineFamilyParityTests(unittest.TestCase):
    def test_the_repository_tables_agree(self):
        self.assertEqual(check.differences(check.go_source(), check.worker_tables()), [])

    def test_the_go_table_is_read_with_its_kinds_and_active_kinds(self):
        families, signed = check.parse_go(check.go_source())
        self.assertEqual(families["nemo"]["kinds"], ["diarization", "fingerprint"])
        self.assertEqual(families["speechbrain"]["active"], ["fingerprint"])
        self.assertEqual(families["wespeaker"]["distribution"], "wespeakerruntime")
        self.assertEqual(signed, {"nemo-titanet", "speechbrain-ecapa"})

    def test_a_family_added_on_one_side_only_is_reported(self):
        source = check.go_source().replace('"wespeaker": {\n\t\tKey: "wespeaker"', '"wespeaker": {\n\t\tKey: "wespeaker"')
        extra = source.replace('var engineFamilies = map[string]engineFamily{',
                               'var engineFamilies = map[string]engineFamily{\n\t"newlib": {\n\t\tKey: "newlib", Engine: "newlib", Distribution: "newlib", Kinds: []string{"fingerprint"},\n\t},')
        self.assertTrue(any("newlib" in d for d in check.differences(extra, check.worker_tables())))

    def test_a_changed_value_is_reported(self):
        source = check.go_source().replace('Distribution: "speechbrain"', 'Distribution: "speechbrain-other"')
        self.assertTrue(any("speechbrain" in d and "distribution" in d for d in check.differences(source, check.worker_tables())))
        source = check.go_source().replace('Kinds: []string{"fingerprint", "separation"}, ActiveKinds: []string{"fingerprint"}',
                                           'Kinds: []string{"fingerprint", "separation"}')
        self.assertTrue(any("active" in d for d in check.differences(source, check.worker_tables())))

    def test_a_signed_only_adapter_changed_on_one_side_is_reported(self):
        source = check.go_source().replace('"speechbrain-ecapa": true}', '"speechbrain-ecapa": true, "other-adapter": true}')
        self.assertTrue(any("other-adapter" in d for d in check.differences(source, check.worker_tables())))


if __name__ == "__main__":
    unittest.main()
