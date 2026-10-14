import json
import tempfile
import unittest
from pathlib import Path

from report import summarize


class ReportTest(unittest.TestCase):
    def test_measurement_excludes_ramp_and_drain_but_stock_includes_them(self):
        points = []
        for phase, api, outcome, status, code, duration in [
            ('ramp','decrement','success','200','OK',999),
            ('measure','search','success','200','OK',10),
            ('measure','decrement','success','200','OK',20),
            ('measure','search','failure','0','TIMEOUT',30),
            ('measure','decrement','business','409','INSUFFICIENT_STOCK',40),
            ('drain','decrement','success','200','OK',999),
        ]:
            tags = dict(phase=phase, api=api, outcome=outcome, status=status, code=code)
            for metric, value in [('http_reqs',1),('load_requests',1),('load_duration_ms',duration)]:
                points.append(dict(type='Point',metric=metric,data=dict(value=value,tags=tags)))
        with tempfile.TemporaryDirectory() as folder:
            prefix = str(Path(folder)/'test')
            Path(prefix+'.config.json').write_text(json.dumps(dict(hold=10,phase='average')))
            Path(prefix+'.json').write_text('\n'.join(json.dumps(p) for p in points))
            report = summarize(prefix)
        self.assertEqual(report['measurement_responses'],4)
        self.assertEqual(report['success_rps'],.2)
        self.assertEqual(report['failure_rate'],.25)
        self.assertEqual(report['business_rejection_rate'],.25)
        self.assertEqual(report['timeout_rate'],.25)
        self.assertEqual(report['successful_decrements_all_phases'],3)
        self.assertTrue(report['completed_requests_match'])
        overall = report['latency_ms'][-1]
        self.assertAlmostEqual(overall['p95'],38.5)
        self.assertAlmostEqual(overall['p99'],39.7)


if __name__ == '__main__':
    unittest.main()
